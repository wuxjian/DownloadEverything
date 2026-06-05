package downloader

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HTTPDownloader HTTP下载器实现
type HTTPDownloader struct {
	threads    int // 分片线程数
	proxyURL   string
	maxRetries int
	retryDelay time.Duration
	mu         sync.RWMutex
	pauses     map[string]context.CancelFunc
	progressFn func(taskID string, p *Progress)
}

// NewHTTPDownloader 创建HTTP下载器
func NewHTTPDownloader(threads int, proxyURL string, maxRetries int, retryInterval int, progressFn func(string, *Progress)) *HTTPDownloader {
	return &HTTPDownloader{
		threads:    threads,
		proxyURL:   proxyURL,
		maxRetries: maxRetries,
		retryDelay: time.Duration(retryInterval) * time.Second,
		pauses:     make(map[string]context.CancelFunc),
		progressFn: progressFn,
	}
}

// buildRequest 构建HTTP请求（注入自定义Header和Cookie）
func (d *HTTPDownloader) buildRequest(ctx context.Context, task *DownloadTask, start, end int64) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", task.URL, nil)
	if err != nil {
		return nil, err
	}

	// 注入默认Header
	for k, v := range DefaultHeaders {
		req.Header.Set(k, v)
	}
	// 覆盖自定义Header
	for k, v := range task.Headers {
		req.Header.Set(k, v)
	}
	// 注入Cookie
	for _, c := range task.Cookies {
		req.AddCookie(c)
	}
	// 断点续传 Range
	if start > 0 || end > 0 {
		if end > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
		} else {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", start))
		}
	}
	return req, nil
}

// httpClient 创建健壮的HTTP客户端（支持代理）
func (d *HTTPDownloader) httpClient() *http.Client {
	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: false},
		TLSHandshakeTimeout: 30 * time.Second,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true, // 不使用gzip，便于断点续传
	}

	// 配置代理
	if d.proxyURL != "" {
		if proxyParsed, err := url.Parse(d.proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(proxyParsed)
			log.Printf("[下载] 使用代理: %s", d.proxyURL)
		} else {
			log.Printf("[下载] 代理地址解析失败: %v", err)
		}
	}

	return &http.Client{
		Timeout:   0, // 无超时，由context控制
		Transport: transport,
	}
}

// doWithRetry 带重试的HTTP请求
// reqBuilder 用于在每次重试时重建请求（避免复用已消耗的 request）
func (d *HTTPDownloader) doWithRetry(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= d.maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避: retryDelay * 2^(attempt-1)
			backoff := d.retryDelay * time.Duration(int(math.Pow(2, float64(attempt-1))))
			if backoff > 5*time.Minute {
				backoff = 5 * time.Minute
			}
			log.Printf("[下载] 第%d次重试，等待%v...", attempt, backoff)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		log.Printf("[下载] 请求失败(第%d次): %v", attempt+1, err)

		// 重建请求（原请求可能被 client.Do 内部修改）
		newReq, rerr := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), nil)
		if rerr != nil {
			return nil, fmt.Errorf("重建请求失败: %w", rerr)
		}
		newReq.Header = req.Header.Clone()
		req = newReq
	}
	return nil, fmt.Errorf("重试%d次后仍失败: %w", d.maxRetries, lastErr)
}

// getFileSize 获取远程文件大小（HEAD失败时降级为GET探测）
// 探测阶段不重试，快速失败后降级为直接下载（下载阶段才重试）
func (d *HTTPDownloader) getFileSize(ctx context.Context, task *DownloadTask) (int64, bool, error) {
	client := d.httpClient()

	// === 第一步：HEAD 探测（15s 超时，不重试） ===
	headReq, err := d.buildRequest(ctx, task, 0, 0)
	if err != nil {
		return 0, false, err
	}
	headReq.Method = "HEAD"

	headCtx, headCancel := context.WithTimeout(ctx, 15*time.Second)
	defer headCancel()
	headReq = headReq.WithContext(headCtx)

	resp, err := client.Do(headReq)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			acceptRange := resp.Header.Get("Accept-Ranges")
			if resp.ContentLength > 0 {
				return resp.ContentLength, acceptRange == "bytes", nil
			}
			if acceptRange == "bytes" {
				return 0, true, nil
			}
		}
	}

	// === 第二步：GET + Range: bytes=0-0 探测（15s 超时，不重试） ===
	log.Printf("[下载] HEAD失败(%v), 尝试GET探测文件大小", err)
	getReq, err := d.buildRequest(ctx, task, 0, 0)
	if err != nil {
		return 0, false, err
	}
	getReq.Header.Set("Range", "bytes=0-0")

	getCtx, getCancel := context.WithTimeout(ctx, 15*time.Second)
	defer getCancel()
	getReq = getReq.WithContext(getCtx)

	resp2, err := client.Do(getReq)
	if err != nil {
		// GET探测也失败，不报错，降级为直接下载
		log.Printf("[下载] GET探测也失败: %v, 降级为直接下载", err)
		return 0, false, nil
	}
	defer resp2.Body.Close()

	if resp2.StatusCode == http.StatusPartialContent {
		cr := resp2.Header.Get("Content-Range")
		var total int64
		fmt.Sscanf(cr, "bytes 0-0/%d", &total)
		if total > 0 {
			return total, true, nil
		}
	}

	if resp2.StatusCode == http.StatusOK {
		return resp2.ContentLength, false, nil
	}
	return 0, false, nil
}

// Download 执行下载（支持分片多线程、断点续传）
func (d *HTTPDownloader) Download(ctx context.Context, task *DownloadTask) error {
	// 确保目录存在
	dir := filepath.Dir(task.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 获取文件大小（失败时降级为未知大小直接下载）
	totalSize, supportRange, err := d.getFileSize(ctx, task)
	if err != nil {
		log.Printf("[下载] 获取文件大小失败，降级为直接下载: %v", err)
		totalSize = 0
		supportRange = false
	}

	// 通知进度回调文件大小
	if d.progressFn != nil {
		d.progressFn(task.ID, &Progress{Total: totalSize, Status: "downloading"})
	}

	// 不支持Range或文件很小，单线程下载
	threads := d.threads
	if !supportRange || totalSize <= 0 || totalSize < 1024*1024 {
		threads = 1
	}

	if threads == 1 {
		return d.downloadSingle(ctx, task, totalSize)
	}
	return d.downloadMulti(ctx, task, totalSize, threads)
}

// downloadSingle 单线程下载
func (d *HTTPDownloader) downloadSingle(ctx context.Context, task *DownloadTask, totalSize int64) error {
	req, err := d.buildRequest(ctx, task, 0, 0)
	if err != nil {
		return err
	}

	client := d.httpClient()
	resp, err := d.doWithRetry(ctx, client, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("下载请求失败: %d", resp.StatusCode)
	}

	// 如果之前没拿到大小，从响应头获取
	if totalSize <= 0 && resp.ContentLength > 0 {
		totalSize = resp.ContentLength
	}

	f, err := os.Create(task.FilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var downloaded int64
	buf := make([]byte, 256*1024)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	lastDownloaded := int64(0)
	lastTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			now := time.Now()
			elapsed := now.Sub(lastTime).Seconds()
			speed := int64(float64(downloaded-lastDownloaded) / elapsed)
			lastDownloaded = downloaded
			lastTime = now
			var progress float64
			if totalSize > 0 {
				progress = float64(downloaded) / float64(totalSize) * 100
				if progress > 100 {
					progress = 100
				}
			}
			if d.progressFn != nil {
				d.progressFn(task.ID, &Progress{
					Downloaded: downloaded,
					Total:      totalSize,
					Speed:      speed,
					Progress:   progress,
					Status:     "downloading",
				})
			}
		default:
			n, err := resp.Body.Read(buf)
			if n > 0 {
				if _, werr := f.Write(buf[:n]); werr != nil {
					return werr
				}
				downloaded += int64(n)
			}
			if err == io.EOF {
				if d.progressFn != nil {
					d.progressFn(task.ID, &Progress{
						Downloaded: downloaded,
						Total:      totalSize,
						Progress:   100,
						Status:     "done",
					})
				}
				return nil
			}
			if err != nil {
				return err
			}
		}
	}
}

// downloadMulti 多线程分片下载
func (d *HTTPDownloader) downloadMulti(ctx context.Context, task *DownloadTask, totalSize int64, threads int) error {
	// 创建文件并预分配空间
	f, err := os.Create(task.FilePath)
	if err != nil {
		return err
	}
	if err := f.Truncate(totalSize); err != nil {
		f.Close()
		return err
	}
	f.Close()

	chunkSize := totalSize / int64(threads)
	var wg sync.WaitGroup
	errCh := make(chan error, threads)

	var downloaded int64
	var mu sync.Mutex
	ticker := time.NewTicker(500 * time.Millisecond)
	done := make(chan struct{})

	// 进度上报协程
	go func() {
		lastDownloaded := int64(0)
		lastTime := time.Now()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				mu.Lock()
				cur := downloaded
				mu.Unlock()
				now := time.Now()
				elapsed := now.Sub(lastTime).Seconds()
				speed := int64(float64(cur-lastDownloaded) / elapsed)
				lastDownloaded = cur
				lastTime = now
				progress := float64(cur) / float64(totalSize) * 100
				if progress > 100 {
					progress = 100
				}
				if d.progressFn != nil {
					d.progressFn(task.ID, &Progress{
						Downloaded: cur,
						Total:      totalSize,
						Speed:      speed,
						Progress:   progress,
						Status:     "downloading",
					})
				}
			}
		}
	}()

	// 子上下文用于暂停
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	d.mu.Lock()
	d.pauses[task.ID] = cancel
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		delete(d.pauses, task.ID)
		d.mu.Unlock()
	}()

	for i := 0; i < threads; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1
		if i == threads-1 {
			end = totalSize - 1
		}
		wg.Add(1)
		go func(idx int, start, end int64) {
			defer wg.Done()
			if err := d.downloadChunk(childCtx, task, start, end, &downloaded, &mu); err != nil {
				errCh <- err
			}
		}(i, start, end)
	}

	wg.Wait()
	close(done)
	ticker.Stop()

	select {
	case err := <-errCh:
		return err
	default:
	}

	if d.progressFn != nil {
		d.progressFn(task.ID, &Progress{
			Downloaded: totalSize,
			Total:      totalSize,
			Progress:   100,
			Status:     "done",
		})
	}
	return nil
}

// downloadChunk 下载一个分片
func (d *HTTPDownloader) downloadChunk(ctx context.Context, task *DownloadTask, start, end int64, total *int64, mu *sync.Mutex) error {
	req, err := d.buildRequest(ctx, task, start, end)
	if err != nil {
		return err
	}

	client := d.httpClient()
	resp, err := d.doWithRetry(ctx, client, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("分片下载失败: %d", resp.StatusCode)
	}

	f, err := os.OpenFile(task.FilePath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Seek(start, 0); err != nil {
		return err
	}

	buf := make([]byte, 256*1024)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			mu.Lock()
			*total += int64(n)
			mu.Unlock()
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// Pause 暂停下载
func (d *HTTPDownloader) Pause(taskID string) error {
	d.mu.RLock()
	cancel, ok := d.pauses[taskID]
	d.mu.RUnlock()
	if ok {
		cancel()
	}
	return nil
}

// Resume 恢复下载（由Manager调用重新发起下载）
func (d *HTTPDownloader) Resume(taskID string) error {
	return nil // Manager处理
}

// Cancel 取消下载
func (d *HTTPDownloader) Cancel(taskID string) error {
	return d.Pause(taskID)
}
