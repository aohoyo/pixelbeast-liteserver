/**
 * 文件管理模块
 *
 * 负责 HTTP 和 FTP 文件浏览、上传、下载、删除、复制、创建目录
 */

import { globalEvents } from '../core/events.js';

/**
 * 格式化文件大小
 * @param {number} bytes - 字节数
 * @returns {string} 格式化后的大小
 */
function formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i];
}

/**
 * 初始化文件面板
 * @param {Object} dependencies - 依赖注入 { state, api, toast, message }
 */
export function initFilesTab({ state, api, toast, message }) {
    console.log('初始化文件面板...');

    // 状态
    let currentPath = '/';
    let currentFtpPath = '/';
    let currentType = 'http'; // 'http' or 'ftp'

    // 加载状态标记
    let httpLoaded = false;
    let ftpLoaded = false;

    // 保存管理器引用
    let httpManager = null;
    let ftpManager = null;

    // 保存依赖引用供闭包使用
    const depsRef = { toast, message };

    /**
     * 初始化 HTTP 文件管理器
     */
    function initHttpFileManager() {
        const manager = createFileManager('http');
        manager.init(); // 初始化事件绑定
        
        // 监听加载事件
        globalEvents.on('files:load', (data) => {
            if (data.type === 'http') {
                manager.loadFiles();
            }
        });

        return manager;
    }

    /**
     * 初始化 FTP 文件管理器
     */
    function initFtpFileManager() {
        const manager = createFileManager('ftp');
        manager.init(); // 初始化事件绑定
        
        // 监听加载事件
        globalEvents.on('files:load', (data) => {
            if (data.type === 'ftp') {
                manager.loadFiles();
            }
        });

        return manager;
    }

    // 创建管理器实例
    httpManager = initHttpFileManager();
    ftpManager = initFtpFileManager();

    // 监听标签页切换事件
    globalEvents.on('tab:switch:files', loadFilesIfNotLoaded);

    // 也监听通用的标签页切换事件
    globalEvents.on('tab:switch:*', (data) => {
        if (data.tabName === 'files') {
            loadFilesIfNotLoaded();
        }
    });

    // 初始加载检查（处理刷新后直接在文件页的情况）
    setTimeout(() => {
        const filesTab = document.getElementById('files');
        if (filesTab && filesTab.classList.contains('active')) {
            console.log('文件标签页已激活，加载文件...');
            loadFilesIfNotLoaded();
        }
    }, 500);

    function loadFilesIfNotLoaded() {
        if (!httpLoaded && httpManager) {
            console.log('加载 HTTP 文件列表...');
            httpManager.loadFiles();
            httpLoaded = true;
        }
        if (!ftpLoaded && ftpManager) {
            console.log('加载 FTP 文件列表...');
            ftpManager.loadFiles();
            ftpLoaded = true;
        }
    }

    /**
     * 创建文件管理器
     * @param {string} type - 类型 ('http' | 'ftp')
     * @returns {Object} 文件管理器对象
     */
    function createFileManager(type) {
        const prefix = type === 'ftp' ? 'ftp-' : '';
        const apiPath = type === 'ftp' ? '/api/ftp/files' : '/api/files';

        return {
            init(autoLoad = true) {
                // 刷新按钮
                const refreshBtn = document.getElementById(`${prefix}refresh-files`);
                refreshBtn?.addEventListener('click', () => this.loadFiles());

                // 新建目录按钮
                const mkdirBtn = document.getElementById(`${prefix}mkdir-btn`);
                mkdirBtn?.addEventListener('click', () => this.showMkdirModal());

                // 上传按钮
                const uploadBtn = document.getElementById(`${prefix}upload-btn`);
                uploadBtn?.addEventListener('click', () => {
                    const fileInput = document.getElementById(`${prefix}file-input`);
                    if (fileInput) {
                        fileInput.value = ''; // 清空以允许重复选择同一文件
                        fileInput.click();
                    }
                });

                // 文件输入
                const fileInput = document.getElementById(`${prefix}file-input`);
                fileInput?.addEventListener('change', (e) => {
                    this.handleFileUpload(e.target.files);
                });

                // 新建目录对话框
                const mkdirConfirm = document.getElementById('mkdir-confirm');
                const mkdirCancel = document.getElementById('mkdir-cancel');
                const mkdirModal = document.getElementById('mkdir-modal');
                const mkdirName = document.getElementById('mkdir-name');

                mkdirConfirm?.addEventListener('click', async () => {
                    const name = mkdirName?.value.trim();
                    if (name) {
                        await this.createDirectory(name);
                        mkdirModal?.classList.remove('active');
                        if (mkdirName) mkdirName.value = '';
                    }
                });

                mkdirCancel?.addEventListener('click', () => {
                    mkdirModal?.classList.remove('active');
                    if (mkdirName) mkdirName.value = '';
                });

                // 点击模态框背景关闭
                mkdirModal?.addEventListener('click', (e) => {
                    if (e.target === mkdirModal) {
                        mkdirModal?.classList.remove('active');
                        if (mkdirName) mkdirName.value = '';
                    }
                });

                // 回车键确认
                mkdirName?.addEventListener('keypress', (e) => {
                    if (e.key === 'Enter') {
                        mkdirConfirm?.click();
                    }
                });
            },

            async loadFiles(path = null) {
                const loadPath = path || (type === 'ftp' ? currentFtpPath : currentPath);

                try {
                    const response = await api.get(`${apiPath}?path=${encodeURIComponent(loadPath)}`);
                    if (response && response.ok) {
                        const data = await api.parseJSON(response);
                        this.renderFiles(data.files || []);
                    } else {
                        toast.error('加载文件列表失败');
                    }
                } catch (error) {
                    console.error('加载文件失败:', error);
                    toast.error('加载文件列表失败');
                }
            },

            renderFiles(files) {
                const fileList = document.getElementById(`${prefix}file-list`);
                const pathEl = document.getElementById(`${prefix}current-path`);

                if (!fileList) return;

                // 更新路径显示
                const current = type === 'ftp' ? currentFtpPath : currentPath;
                if (pathEl) pathEl.textContent = current;

                // 清空列表
                fileList.innerHTML = '';

                // 添加父目录
                if (current !== '/') {
                    const parentItem = document.createElement('div');
                    parentItem.className = 'file-item parent-dir';
                    parentItem.innerHTML = `
                        <div class="file-name">.. 父目录</div>
                    `;
                    parentItem.addEventListener('click', () => {
                        this.navigateToParent();
                    });
                    fileList.appendChild(parentItem);
                }

                // 渲染文件列表
                for (const file of files) {
                    const item = document.createElement('div');
                    item.className = 'file-item';

                    const icon = file.is_dir ? `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>` : `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>`;
                    const size = file.is_dir ? '' : formatSize(file.size);

                    item.innerHTML = `
                        <div class="file-name"><span class="file-icon">${icon}</span> ${this.escapeHtml(file.name)}</div>
                        <div class="file-size">${size}</div>
                        <div class="file-actions">
                            ${!file.is_dir ? `<button class="download-btn" data-name="${this.escapeHtml(file.name)}" title="下载文件"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg></button>` : ''}
                            <button class="rename-btn" data-name="${this.escapeHtml(file.name)}" title="重命名"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg></button>
                            <button class="copy-btn" data-name="${this.escapeHtml(file.name)}" title="复制"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg></button>
                            <button class="delete-btn" data-name="${this.escapeHtml(file.name)}" title="删除"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg></button>
                        </div>
                    `;

                    // 点击导航
                    item.addEventListener('click', (e) => {
                        if (!e.target.closest('.file-actions')) {
                            if (file.is_dir) {
                                this.navigateTo(file.name);
                            }
                        }
                    });

                    // 下载按钮
                    const downloadBtn = item.querySelector('.download-btn');
                    downloadBtn?.addEventListener('click', async (e) => {
                        e.stopPropagation();
                        await this.downloadFile(file.name);
                    });

                    // 重命名按钮
                    const renameBtn = item.querySelector('.rename-btn');
                    renameBtn?.addEventListener('click', async (e) => {
                        e.stopPropagation();
                        await this.renameFile(file.name);
                    });

                    // 复制按钮
                    const copyBtn = item.querySelector('.copy-btn');
                    copyBtn?.addEventListener('click', async (e) => {
                        e.stopPropagation();
                        await this.copyFile(file.name);
                    });

                    // 删除按钮
                    const deleteBtn = item.querySelector('.delete-btn');
                    deleteBtn?.addEventListener('click', async (e) => {
                        e.stopPropagation();
                        if (confirm(`确定要删除 "${file.name}" 吗？`)) {
                            await this.deleteFile(file.name, file.is_dir);
                        }
                    });

                    fileList.appendChild(item);
                }

                // 如果没有文件
                if (files.length === 0) {
                    const emptyMsg = document.createElement('div');
                    emptyMsg.className = 'file-list-empty';
                    emptyMsg.textContent = '此目录为空';
                    fileList.appendChild(emptyMsg);
                }
            },

            navigateTo(dirname) {
                const current = type === 'ftp' ? currentFtpPath : currentPath;
                let newPath;

                if (current === '/') {
                    newPath = '/' + dirname;
                } else {
                    newPath = current + '/' + dirname;
                }

                // 规范化路径
                newPath = newPath.replace(/\/+/g, '/');

                if (type === 'ftp') {
                    currentFtpPath = newPath;
                } else {
                    currentPath = newPath;
                }

                this.loadFiles(newPath);
            },

            navigateToParent() {
                const current = type === 'ftp' ? currentFtpPath : currentPath;

                if (current === '/') return;

                const parts = current.split('/').filter(Boolean);
                parts.pop();
                const parentPath = parts.length > 0 ? '/' + parts.join('/') : '/';

                if (type === 'ftp') {
                    currentFtpPath = parentPath;
                } else {
                    currentPath = parentPath;
                }

                this.loadFiles(parentPath);
            },

            showMkdirModal() {
                const modal = document.getElementById('mkdir-modal');
                modal?.classList.add('active');
                document.getElementById('mkdir-name')?.focus();
            },

            async createDirectory(name) {
                const current = type === 'ftp' ? currentFtpPath : currentPath;

                try {
                    const response = await api.post(`${apiPath}/mkdir`, {
                        path: current,
                        name: name
                    });

                    const data = response ? await api.parseJSON(response) : null;
                    if (data) {
                        depsRef.message.success('目录创建成功');
                        this.loadFiles();
                    } else {
                        depsRef.message.error('目录创建失败');
                    }
                } catch (error) {
                    console.error('创建目录失败:', error);
                    depsRef.message.error(error.message || '目录创建失败');
                }
            },

            async deleteFile(name, isDir) {
                const current = type === 'ftp' ? currentFtpPath : currentPath;

                try {
                    const response = await api.post(`${apiPath}/delete`, {
                        path: current,
                        name: name
                    });

                    const data = response ? await api.parseJSON(response) : null;
                    if (data) {
                        toast.success(isDir ? '目录删除成功' : '文件删除成功', 2000);
                        this.loadFiles();
                    } else {
                        toast.error(isDir ? '目录删除失败' : '文件删除失败', 3000);
                    }
                } catch (error) {
                    console.error('删除失败:', error);
                    toast.error(error.message || (isDir ? '目录删除失败' : '文件删除失败'), 3000);
                }
            },

            async downloadFile(name) {
                const current = type === 'ftp' ? currentFtpPath : currentPath;

                try {
                    // 获取当前页面路径（管理面板基础路径）
                    const basePath = window.location.pathname.replace(/\/(index\.html)?$/, '') || '';

                    // 构建下载 URL
                    const downloadUrl = `${basePath}${apiPath}/download?path=${encodeURIComponent(current)}&name=${encodeURIComponent(name)}`;

                    // 创建隐藏的 a 标签触发下载
                    const link = document.createElement('a');
                    link.href = downloadUrl;
                    link.download = name;
                    document.body.appendChild(link);
                    link.click();
                    document.body.removeChild(link);

                    toast.success('开始下载', 2000);
                } catch (error) {
                    console.error('下载失败:', error);
                    toast.error('下载失败', 3000);
                }
            },

            async renameFile(name) {
                const newName = prompt('请输入新文件名:', name);
                if (!newName || newName === name) return;

                const current = type === 'ftp' ? currentFtpPath : currentPath;

                try {
                    const response = await api.post(`${apiPath}/rename`, {
                        path: current,
                        oldName: name,
                        newName: newName
                    });

                    const data = response ? await api.parseJSON(response) : null;
                    if (data) {
                        toast.success('重命名成功', 2000);
                        this.loadFiles();
                    } else {
                        toast.error('重命名失败', 3000);
                    }
                } catch (error) {
                    console.error('重命名失败:', error);
                    toast.error(error.message || '重命名失败', 3000);
                }
            },

            async copyFile(name) {
                const current = type === 'ftp' ? currentFtpPath : currentPath;

                // 构建文件访问链接
                let fileUrl = '';
                if (type === 'http') {
                    // HTTP 文件：构建完整 URL
                    const protocol = window.location.protocol;
                    const host = window.location.host;

                    // 构建文件路径（相对于 HTTP 根目录）
                    let filePath = current === '/' ? name : current.replace(/^\//, '') + '/' + name;

                    // 构建完整 URL
                    fileUrl = `${protocol}//${host}/${filePath}`;

                    // 复制链接到剪贴板
                    try {
                        await navigator.clipboard.writeText(fileUrl);
                        depsRef.message.success('链接已复制');
                    } catch (err) {
                        // 降级方案：使用传统方法
                        const textArea = document.createElement('textarea');
                        textArea.value = fileUrl;
                        textArea.style.position = 'fixed';
                        textArea.style.opacity = '0';
                        document.body.appendChild(textArea);
                        textArea.select();
                        try {
                            document.execCommand('copy');
                            depsRef.message.success('链接已复制');
                        } catch (e) {
                            depsRef.message.error('复制失败，请手动复制链接');
                            console.log('文件链接:', fileUrl);
                        }
                        document.body.removeChild(textArea);
                    }
                } else {
                    // FTP 文件：执行文件复制功能
                    // 生成新文件名
                    let newName = name;
                    const extIndex = name.lastIndexOf('.');
                    if (extIndex > 0) {
                        const baseName = name.substring(0, extIndex);
                        const ext = name.substring(extIndex);
                        newName = `${baseName}-copy${ext}`;
                    } else {
                        newName = `${name}-copy`;
                    }

                    try {
                        const response = await api.post(`${apiPath}/copy`, {
                            srcPath: current,
                            srcName: name,
                            dstName: newName
                        });

                        const data = response ? await api.parseJSON(response) : null;
                        if (data) {
                            toast.success('文件复制成功', 2000);
                            this.loadFiles();
                        } else {
                            toast.error('文件复制失败', 3000);
                        }
                    } catch (error) {
                        console.error('复制失败:', error);
                        toast.error(error.message || '文件复制失败', 3000);
                    }
                }
            },

            async handleFileUpload(files) {
                if (files.length === 0) return;

                const current = type === 'ftp' ? currentFtpPath : currentPath;

                // 批量上传：显示总进度
                const totalFiles = files.length;
                let uploadedCount = 0;
                let failedFiles = [];

                const progressContainer = document.getElementById(`${prefix}upload-progress`);
                const progressText = document.getElementById(`${prefix}progress-text`);
                const progressPercent = document.getElementById(`${prefix}progress-percent`);
                const progressFill = document.getElementById(`${prefix}progress-fill`);

                progressContainer?.style.removeProperty('display');

                for (const file of files) {
                    try {
                        await this.uploadFile(file, current, (percent) => {
                            if (progressText) {
                                progressText.textContent = `正在上传 ${file.name} (${uploadedCount + 1}/${totalFiles})...`;
                            }
                            if (progressPercent) {
                                const overallPercent = Math.round(((uploadedCount * 100) + percent) / totalFiles);
                                progressPercent.textContent = `${overallPercent}%`;
                            }
                            if (progressFill) {
                                const overallPercent = ((uploadedCount * 100) + percent) / totalFiles;
                                progressFill.style.width = `${overallPercent}%`;
                            }
                        });
                        uploadedCount++;
                    } catch (error) {
                        console.error(`上传失败: ${file.name}`, error);
                        failedFiles.push(file.name);
                    }
                }

                // 隐藏进度条
                if (progressContainer) progressContainer.style.display = 'none';
                if (progressFill) progressFill.style.width = '0%';

                // 显示结果
                if (failedFiles.length === 0) {
                    depsRef.message.success(`成功上传 ${uploadedCount} 个文件`);
                } else {
                    depsRef.message.error(`上传完成：成功 ${uploadedCount} 个，失败 ${failedFiles.length} 个`);
                }

                // 刷新文件列表
                this.loadFiles();
            },

            async uploadFile(file, path, onProgress) {
                const formData = new FormData();
                formData.append('file', file);
                formData.append('path', path);

                await api.upload(`${apiPath}/upload`, formData, onProgress);
            },

            escapeHtml(text) {
                const div = document.createElement('div');
                div.textContent = text;
                return div.innerHTML;
            }
        };
    }
}
