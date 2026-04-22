/**
 * UploadManager - 文件上传管理器
 *
 * 支持直接上传、分块上传、断点续传
 * 自动判断文件大小选择上传方式
 */

import { getCSRFToken } from '../core/api.js';

const CHUNK_SIZE = 5 * 1024 * 1024;           // 5MB per chunk
const LARGE_FILE_THRESHOLD = 50 * 1024 * 1024; // > 50MB 使用分块上传
const MAX_CONCURRENT = 3;

function fetchWithCSRF(url, options = {}) {
    if (!options.headers) options.headers = {};
    const token = getCSRFToken();
    if (token) options.headers['X-CSRF-Token'] = token;
    return fetch(url, options);
}

export class UploadManager {
    constructor(options) {
        this.apiPath = options.apiPath || '/admin/api/files';
        this.onProgress = options.onProgress || (() => {});
        this.onFileComplete = options.onFileComplete || (() => {});
        this.onAllComplete = options.onAllComplete || (() => {});
        this.onError = options.onError || (() => {});

        this.activeUploads = new Map();    // uploadID -> info
        this.abortControllers = new Map(); // uploadID -> AbortController
        this.speedTrackers = new Map();    // uploadID -> { lastTime, lastLoaded, speed }
    }

    /**
     * 批量上传文件
     * @param {File[]} files
     * @param {string} destPath
     * @param {Object} options - { relativePaths: Map<File, string> }
     */
    async uploadFiles(files, destPath, options = {}) {
        const { relativePaths } = options;
        const tasks = [];

        for (const file of files) {
            const relativePath = relativePaths?.get(file) || '';
            if (file.size > LARGE_FILE_THRESHOLD) {
                tasks.push(() => this.uploadChunked(file, destPath, relativePath));
            } else {
                tasks.push(() => this.uploadDirect(file, destPath, relativePath));
            }
        }

        await this.runConcurrent(tasks, MAX_CONCURRENT);
        this.onAllComplete();
    }

    /**
     * 小文件直接上传（XHR 带进度）
     */
    async uploadDirect(file, destPath, relativePath = '') {
        const uploadID = this.generateUploadID(file, destPath);
        const abortController = new AbortController();
        this.abortControllers.set(uploadID, abortController);

        this.activeUploads.set(uploadID, {
            file, fileName: file.name, destPath, relativePath,
            progress: 0, status: 'uploading', type: 'direct'
        });
        this.onProgress(this.getSnapshot());

        try {
            const formData = new FormData();
            formData.append('file', file);
            formData.append('path', destPath);
            if (relativePath) formData.append('relativePath', relativePath);

            await new Promise((resolve, reject) => {
                const token = getCSRFToken();
                const xhr = new XMLHttpRequest();
                xhr.open('POST', `${this.apiPath}/upload/path`);
                if (token) xhr.setRequestHeader('X-CSRF-Token', token);

                xhr.upload.onprogress = (e) => {
                    if (e.lengthComputable) {
                        this.trackSpeed(uploadID, e.loaded);
                        this.throttledUpdate(uploadID, Math.round((e.loaded / e.total) * 100));
                    }
                };
                xhr.onload = () => {
                    if (xhr.status >= 200 && xhr.status < 300) resolve();
                    else reject(new Error(`HTTP ${xhr.status}`));
                };
                xhr.onerror = () => reject(new Error('网络错误'));
                xhr.onabort = () => reject(new Error('已取消'));

                abortController.signal.addEventListener('abort', () => xhr.abort());
                xhr.send(formData);
            });

            this.setStatus(uploadID, 100, 'complete');
            this.onFileComplete(file.name);
        } catch (error) {
            this.setStatus(uploadID, undefined, error.message === '已取消' ? 'paused' : 'error');
            if (error.message !== '已取消') {
                this.onError(file.name, error.message);
            }
        } finally {
            this.abortControllers.delete(uploadID);
        }
    }

    /**
     * 分块上传（支持断点续传）
     */
    async uploadChunked(file, destPath, relativePath = '') {
        const uploadID = this.generateUploadID(file, destPath);
        const totalChunks = Math.ceil(file.size / CHUNK_SIZE);
        const abortController = new AbortController();
        this.abortControllers.set(uploadID, abortController);

        this.activeUploads.set(uploadID, {
            file, fileName: file.name, destPath, relativePath,
            progress: 0, status: 'uploading', type: 'chunked',
            totalChunks
        });
        this.onProgress(this.getSnapshot());

        try {
            // 1. 检查已有分块（断点续传）
            const existingChunks = await this.checkExistingChunks(uploadID);
            let uploadedCount = existingChunks.length;

            // 更新进度
            if (uploadedCount > 0) {
                this.updateProgress(uploadID, Math.round((uploadedCount / totalChunks) * 100));
            }

            // 2. 上传缺失的分块
            for (let i = 0; i < totalChunks; i++) {
                if (existingChunks.includes(i)) continue;
                if (abortController.signal.aborted) throw new Error('已取消');

                const start = i * CHUNK_SIZE;
                const end = Math.min(start + CHUNK_SIZE, file.size);
                const chunkBlob = file.slice(start, end);

                const formData = new FormData();
                formData.append('chunk', chunkBlob);
                formData.append('uploadID', uploadID);
                formData.append('chunkIndex', String(i));

                const resp = await fetchWithCSRF(`${this.apiPath}/upload/chunk`, {
                    method: 'POST', body: formData,
                    signal: abortController.signal
                });
                if (!resp.ok) throw new Error(`分块上传失败: HTTP ${resp.status}`);

                uploadedCount++;
                this.updateProgress(uploadID, Math.round((uploadedCount / totalChunks) * 100));
            }

            // 3. 合并分块
            const mergeResp = await fetchWithCSRF(`${this.apiPath}/upload/merge`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    uploadID,
                    filename: relativePath || file.name,
                    totalChunks,
                    destPath
                })
            });
            if (!mergeResp.ok) throw new Error('合并失败');

            this.setStatus(uploadID, 100, 'complete');
            this.onFileComplete(file.name);
        } catch (error) {
            this.setStatus(uploadID, undefined, error.message === '已取消' ? 'paused' : 'error');
            if (error.message !== '已取消') {
                this.onError(file.name, error.message);
            }
        } finally {
            this.abortControllers.delete(uploadID);
        }
    }

    /**
     * 查询已上传的分块索引
     */
    async checkExistingChunks(uploadID) {
        try {
            const resp = await fetch(`${this.apiPath}/upload/status?uploadID=${encodeURIComponent(uploadID)}`);
            const result = await resp.json();
            if (result?.code === 200 && result.data?.chunks) return result.data.chunks;
        } catch (e) { /* ignore */ }
        return [];
    }

    /**
     * 暂停上传
     */
    pauseUpload(uploadID) {
        this.abortControllers.get(uploadID)?.abort();
    }

    /**
     * 取消上传（暂停 + 从列表移除）
     */
    removeUpload(uploadID) {
        this.abortControllers.get(uploadID)?.abort();
        this.activeUploads.delete(uploadID);
        this.onProgress(this.getSnapshot());
    }

    /**
     * 恢复上传（重新触发分块上传，自动跳过已有分块）
     */
    async resumeUpload(uploadID) {
        const info = this.activeUploads.get(uploadID);
        if (!info || info.type !== 'chunked') return;
        this.setStatus(uploadID, info.progress || 0, 'uploading');
        await this.uploadChunked(info.file, info.destPath, info.relativePath || '');
    }

    // ========== 工具方法 ==========

    trackSpeed(uploadID, loaded) {
        const now = Date.now();
        const tracker = this.speedTrackers.get(uploadID) || { lastTime: now, lastLoaded: 0, speed: 0 };
        const elapsed = (now - tracker.lastTime) / 1000;
        if (elapsed > 0.3) {
            tracker.speed = Math.round((loaded - tracker.lastLoaded) / elapsed);
            tracker.lastTime = now;
            tracker.lastLoaded = loaded;
            this.speedTrackers.set(uploadID, tracker);
        }
    }

    throttledUpdate(uploadID, progress) {
        const now = Date.now();
        const lastUpdate = this._lastUIUpdate || 0;
        const info = this.activeUploads.get(uploadID);
        if (info) {
            const tracker = this.speedTrackers.get(uploadID);
            this.activeUploads.set(uploadID, {
                ...info, progress,
                speed: tracker?.speed || 0
            });
        }
        if (now - lastUpdate > 300) { // 每 300ms 更新一次 UI
            this._lastUIUpdate = now;
            this.onProgress(this.getSnapshot());
        }
    }

    formatSpeed(speed) {
        if (!speed || speed <= 0) return '';
        if (speed < 1024) return speed + ' B/s';
        if (speed < 1024 * 1024) return (speed / 1024).toFixed(1) + ' KB/s';
        return (speed / (1024 * 1024)).toFixed(1) + ' MB/s';
    }

    generateUploadID(file, destPath) {
        const raw = `${file.name}-${file.size}-${file.lastModified}-${destPath}`;
        let hash = 0;
        for (let i = 0; i < raw.length; i++) {
            hash = ((hash << 5) - hash) + raw.charCodeAt(i);
            hash = hash & hash;
        }
        return `${Math.abs(hash).toString(36)}-${file.size}`;
    }

    updateProgress(uploadID, progress) {
        const info = this.activeUploads.get(uploadID);
        if (info) {
            this.activeUploads.set(uploadID, { ...info, progress });
            this.onProgress(this.getSnapshot());
        }
    }

    setStatus(uploadID, progress, status) {
        const info = this.activeUploads.get(uploadID);
        if (info) {
            this.activeUploads.set(uploadID, {
                ...info,
                ...(progress !== undefined ? { progress } : {}),
                status
            });
            this.onProgress(this.getSnapshot());
        }
    }

    getSnapshot() {
        return Array.from(this.activeUploads.entries()).map(([id, info]) => ({ id, ...info }));
    }

    async runConcurrent(taskFns, max) {
        const executing = new Set();
        for (const fn of taskFns) {
            const p = fn().then(() => { executing.delete(p); });
            executing.add(p);
            if (executing.size >= max) await Promise.race(executing);
        }
        await Promise.all(executing);
    }
}
