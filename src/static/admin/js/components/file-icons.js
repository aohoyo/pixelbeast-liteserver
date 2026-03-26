/**
 * 文件图标 - 轻量级SVG图标方案
 * 
 * 使用 SVG sprite 或内联 SVG，无需外部依赖
 */

// 文件类型图标映射
const FILE_ICONS = {
    // 文件夹
    folder: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" fill="currentColor" fill-opacity="0.2"/></svg>`,
    folderOpen: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2v1M2 10h20" stroke-linecap="round"/></svg>`,
    
    // 文档类型
    file: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>`,
    txt: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>`,
    doc: `<svg viewBox="0 0 24 24" fill="none" stroke="#2196F3" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><text x="7" y="16" font-size="6" fill="#2196F3">DOC</text></svg>`,
    
    // 图片类型
    image: `<svg viewBox="0 0 24 24" fill="none" stroke="#4CAF50" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5" fill="#4CAF50"/><path d="M21 15l-5-5L5 21" stroke-linecap="round"/></svg>`,
    
    // 视频类型
    video: `<svg viewBox="0 0 24 24" fill="none" stroke="#E91E63" stroke-width="1.5"><rect x="2" y="4" width="20" height="16" rx="2"/><polygon points="10 8 16 12 10 16" fill="#E91E63"/></svg>`,
    
    // 音频类型
    audio: `<svg viewBox="0 0 24 24" fill="none" stroke="#9C27B0" stroke-width="1.5"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3" fill="#9C27B0"/><circle cx="18" cy="16" r="3" fill="#9C27B0"/></svg>`,
    
    // 压缩包
    archive: `<svg viewBox="0 0 24 24" fill="none" stroke="#FF9800" stroke-width="1.5"><path d="M21 8v13H3V8"/><path d="M3 8l4-4h10l4 4" fill="#FF9800" fill-opacity="0.2"/><rect x="10" y="4" width="4" height="4"/><line x1="12" y1="8" x2="12" y2="10"/><line x1="12" y1="10" x2="10" y2="10"/><line x1="12" y1="12" x2="14" y2="12"/></svg>`,
    
    // 代码
    code: `<svg viewBox="0 0 24 24" fill="none" stroke="#00BCD4" stroke-width="1.5"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>`,
    
    // PDF
    pdf: `<svg viewBox="0 0 24 24" fill="none" stroke="#F44336" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><text x="6" y="16" font-size="5" fill="#F44336">PDF</text></svg>`,
    
    // 配置文件
    config: `<svg viewBox="0 0 24 24" fill="none" stroke="#607D8B" stroke-width="1.5"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06a1.65 1.65 0 00.33-1.82 1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06a1.65 1.65 0 001.82.33H9a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06a1.65 1.65 0 00-.33 1.82V9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z"/></svg>`
};

// 文件扩展名到图标类型映射
const EXT_MAP = {
    // 图片
    'jpg': 'image', 'jpeg': 'image', 'png': 'image', 'gif': 'image', 
    'webp': 'image', 'svg': 'image', 'bmp': 'image', 'ico': 'image',
    // 视频
    'mp4': 'video', 'mkv': 'video', 'avi': 'video', 'mov': 'video',
    'wmv': 'video', 'flv': 'video', 'webm': 'video',
    // 音频
    'mp3': 'audio', 'wav': 'audio', 'flac': 'audio', 'aac': 'audio',
    'ogg': 'audio', 'wma': 'audio', 'm4a': 'audio',
    // 压缩包
    'zip': 'archive', 'rar': 'archive', '7z': 'archive', 'tar': 'archive',
    'gz': 'archive', 'bz2': 'archive',
    // 代码
    'js': 'code', 'ts': 'code', 'jsx': 'code', 'tsx': 'code',
    'html': 'code', 'css': 'code', 'scss': 'code', 'less': 'code',
    'json': 'code', 'xml': 'code', 'yaml': 'code', 'yml': 'code',
    'py': 'code', 'rb': 'code', 'go': 'code', 'rs': 'code',
    'java': 'code', 'c': 'code', 'cpp': 'code', 'h': 'code',
    'sh': 'code', 'bat': 'code', 'ps1': 'code',
    // 文档
    'pdf': 'pdf', 'doc': 'doc', 'docx': 'doc',
    'txt': 'txt', 'md': 'txt', 'rtf': 'txt',
    // 配置
    'conf': 'config', 'cfg': 'config', 'ini': 'config',
    'env': 'config', 'toml': 'config'
};

/**
 * 获取文件图标
 * @param {string} name - 文件名
 * @param {boolean} isDir - 是否为目录
 * @returns {string} SVG 图标 HTML
 */
export function getFileIcon(name, isDir = false) {
    if (isDir) return FILE_ICONS.folder;
    
    const ext = name.split('.').pop()?.toLowerCase() || '';
    const type = EXT_MAP[ext] || 'file';
    return FILE_ICONS[type] || FILE_ICONS.file;
}

/**
 * 获取图标颜色类
 * @param {string} name - 文件名
 * @param {boolean} isDir - 是否为目录
 * @returns {string} CSS 类名
 */
export function getIconColorClass(name, isDir = false) {
    if (isDir) return 'file-icon-folder';
    
    const ext = name.split('.').pop()?.toLowerCase() || '';
    const type = EXT_MAP[ext] || 'file';
    
    const colorMap = {
        'image': 'file-icon-image',
        'video': 'file-icon-video',
        'audio': 'file-icon-audio',
        'archive': 'file-icon-archive',
        'code': 'file-icon-code',
        'pdf': 'file-icon-pdf',
        'doc': 'file-icon-doc'
    };
    
    return colorMap[type] || 'file-icon-default';
}

/**
 * 格式化文件大小
 */
export function formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    if (bytes === null || bytes === undefined) return '-';
    
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return (bytes / Math.pow(k, i)).toFixed(1) + ' ' + sizes[i];
}

/**
 * 格式化日期
 */
export function formatDate(timestamp) {
    if (!timestamp) return '-';
    const d = new Date(timestamp);
    return d.toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

export default { getFileIcon, getIconColorClass, formatFileSize, formatDate };