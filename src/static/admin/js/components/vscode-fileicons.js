/**
 * VSCode Material Icon Theme 风格文件图标
 *
 * 灵感来源: vscode-material-icon-theme
 * 使用内联 SVG，无需外部字体/CSS 依赖
 */

// ==================== SVG 图标定义 ====================
// 文件形状基础 + 语言徽标，模仿 VSCode Material Icon Theme 风格

const FILE_BASE = `<path d="M13.7 2H6.3C5.6 2 5 2.6 5 3.3v17.4c0 .7.6 1.3 1.3 1.3h11.4c.7 0 1.3-.6 1.3-1.3V7.5L13.7 2z" fill="#90A4AE"/>`;
const FILE_FOLD = `<path d="M13.7 2v4.2c0 .7.6 1.3 1.3 1.3H19L13.7 2z" fill="#B0BEC5" opacity="0.6"/>`;
const FOLDER_PATH = `<path d="M10 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z" fill="#FFA726"/>`;
const FOLDER_OPEN_PATH = `<path d="M20 6h-8l-2-2H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2z" fill="#FFA726"/><path d="M2 10h20v8c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2v-8z" fill="#FFB74D" opacity="0.5"/>`;

// 创建文件图标的辅助函数
function fileIcon(content, color) {
	return `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">${FILE_BASE}${FILE_FOLD}${content}</svg>`;
}

// 创建带底部徽标的文件图标
function badgeFileIcon(label, bgColor, textColor = '#fff') {
	return fileIcon(
		`<rect x="7" y="12.5" width="11" height="7.5" rx="1.2" fill="${bgColor}"/>` +
		`<text x="12.5" y="18.2" font-size="4.5" fill="${textColor}" text-anchor="middle" font-weight="700" font-family="sans-serif">${label}</text>`
	);
}

// ==================== 图标集合 ====================

const ICONS = {
	// 文件夹
	folder: `<svg viewBox="0 0 24 24" fill="none">${FOLDER_PATH}</svg>`,
	folderOpen: `<svg viewBox="0 0 24 24" fill="none">${FOLDER_OPEN_PATH}</svg>`,

	// 通用文件
	file: fileIcon('', '#90A4AE'),

	// ==================== 编程语言 ====================
	go: badgeFileIcon('GO', '#00ADD8'),
	javascript: badgeFileIcon('JS', '#FBC02D', '#323330'),
	typescript: badgeFileIcon('TS', '#3178C6'),
	jsx: badgeFileIcon('JSX', '#00B8D4'),
	tsx: badgeFileIcon('TSX', '#3178C6'),
	python: fileIcon(
		`<rect x="7" y="12.5" width="11" height="7.5" rx="1.2" fill="#3776AB"/>` +
		`<text x="12.5" y="18.2" font-size="4.5" fill="#FFD43B" text-anchor="middle" font-weight="700" font-family="sans-serif">Py</text>`
	),
	java: badgeFileIcon('J', '#E76F00'),
	c: badgeFileIcon('C', '#555555'),
	cpp: badgeFileIcon('C++', '#00599C'),
	h: badgeFileIcon('H', '#8E24AA'),
	hpp: badgeFileIcon('HP', '#8E24AA'),
	rust: badgeFileIcon('RS', '#CE412B'),
	ruby: badgeFileIcon('RB', '#CC342D'),
	php: badgeFileIcon('PHP', '#777BB4'),
	swift: badgeFileIcon('SW', '#FA7343'),
	kotlin: badgeFileIcon('K', '#7F52FF'),
	dart: badgeFileIcon('DAR', '#0175C2'),
	lua: badgeFileIcon('LUA', '#000080'),
	perl: badgeFileIcon('PL', '#39457E'),
	r: badgeFileIcon('R', '#276DC3'),
	cs: badgeFileIcon('C#', '#68217A'),

	// ==================== Web 技术 ====================
	html: fileIcon(
		`<rect x="7" y="12.5" width="11" height="7.5" rx="1.2" fill="#E44D26"/>` +
		`<text x="12.5" y="18.2" font-size="4" fill="#fff" text-anchor="middle" font-weight="700" font-family="sans-serif">&lt;/&gt;</text>`
	),
	css: badgeFileIcon('CS', '#1572B6'),
	scss: badgeFileIcon('S', '#CC6699'),
	sass: badgeFileIcon('S', '#CC6699'),
	less: badgeFileIcon('LES', '#1D365D'),
	vue: badgeFileIcon('V', '#4FC08D'),
	svelte: badgeFileIcon('SV', '#FF3E00'),
	react: badgeFileIcon('RE', '#61DAFB', '#282C34'),

	// ==================== 数据/配置 ====================
	json: fileIcon(
		`<rect x="7" y="12.5" width="11" height="7.5" rx="1.2" fill="#FBC02D"/>` +
		`<text x="12.5" y="18.2" font-size="5" fill="#323330" text-anchor="middle" font-weight="700" font-family="sans-serif">{}</text>`
	),
	xml: badgeFileIcon('XML', '#E44D26'),
	yaml: badgeFileIcon('YM', '#CB171E'),
	toml: badgeFileIcon('TOM', '#9C4221'),
	sql: fileIcon(
		`<ellipse cx="12" cy="14" rx="5.5" ry="2.5" fill="#0078D4"/>` +
		`<path d="M6.5 14v4c0 1.4 2.5 2.5 5.5 2.5s5.5-1.1 5.5-2.5v-4" fill="none" stroke="#0078D4" stroke-width="1"/>` +
		`<ellipse cx="12" cy="14" rx="5.5" ry="2.5" fill="none" stroke="#0078D4" stroke-width="1"/>`
	),

	// ==================== 配置文件 ====================
	config: fileIcon(
		`<path d="M12 15.5A3.5 3.5 0 1012 8.5a3.5 3.5 0 000 7z" fill="#90A4AE"/>` +
		`<path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 11-2.83 2.83l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09a1.65 1.65 0 00-1.08-1.51 1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06a1.65 1.65 0 00.33-1.82 1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09a1.65 1.65 0 001.51-1.08 1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 112.83-2.83l.06.06a1.65 1.65 0 001.82.33H9a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 112.83 2.83l-.06.06a1.65 1.65 0 00-.33 1.82V9c.26.6.83 1 1.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z" fill="none" stroke="#90A4AE" stroke-width="0.8" opacity="0.4"/>`
	),

	// ==================== 文档 ====================
	text: fileIcon(
		`<line x1="8" y1="13" x2="16" y2="13" stroke="#78909C" stroke-width="1"/>` +
		`<line x1="8" y1="15.5" x2="14" y2="15.5" stroke="#78909C" stroke-width="1"/>` +
		`<line x1="8" y1="18" x2="16" y2="18" stroke="#78909C" stroke-width="1"/>`
	),
	markdown: badgeFileIcon('M↓', '#519ABA'),
	pdf: badgeFileIcon('PDF', '#E53935'),
	doc: badgeFileIcon('W', '#2B579A'),
	rtf: badgeFileIcon('RTF', '#2B579A'),
	log: badgeFileIcon('LOG', '#78909C'),

	// ==================== 图片/音视频 ====================
	image: fileIcon(
		`<rect x="7" y="12.5" width="11" height="7.5" rx="1.2" fill="#43A047"/>` +
		`<circle cx="10" cy="15" r="1.2" fill="#A5D6A7"/>` +
		`<path d="M7 19l3.5-3.5 2 2 3-3L18 19" fill="#A5D6A7"/>`
	),
	video: fileIcon(
		`<rect x="7" y="13" width="11" height="7" rx="1.2" fill="#E91E63"/>` +
		`<polygon points="10.5,14.5 10.5,18.5 14.5,16.5" fill="#fff" opacity="0.9"/>`
	),
	audio: fileIcon(
		`<rect x="7" y="12.5" width="11" height="7.5" rx="1.2" fill="#7B1FA2"/>` +
		`<path d="M11 14v3.5a1.5 1.5 0 11-1-1.4V14l4-1v4a1.5 1.5 0 11-1-1.4V13l-2 .5z" fill="#CE93D8"/>`
	),
	svg: badgeFileIcon('SVG', '#FFB13B'),

	// ==================== 压缩包 ====================
	archive: fileIcon(
		`<rect x="7" y="12.5" width="11" height="7.5" rx="1.2" fill="#FF8F00"/>` +
		`<rect x="10" y="12.5" width="4" height="2" fill="#FFC107"/>` +
		`<rect x="11" y="14.5" width="2" height="1" fill="#FFC107"/>` +
		`<rect x="11" y="16" width="2" height="1" fill="#FFC107"/>` +
		`<rect x="11" y="17.5" width="2" height="1.5" fill="#FFC107"/>`
	),

	// ==================== 其他 ====================
	docker: fileIcon(
		`<rect x="7" y="12.5" width="11" height="7.5" rx="1.2" fill="#2496ED"/>` +
		`<rect x="8.5" y="15.5" width="2" height="1.5" rx="0.3" fill="#fff" opacity="0.8"/>` +
		`<rect x="11" y="15.5" width="2" height="1.5" rx="0.3" fill="#fff" opacity="0.8"/>` +
		`<rect x="13.5" y="15.5" width="2" height="1.5" rx="0.3" fill="#fff" opacity="0.8"/>` +
		`<rect x="11" y="13.5" width="2" height="1.5" rx="0.3" fill="#fff" opacity="0.8"/>` +
		`<rect x="13.5" y="13.5" width="2" height="1.5" rx="0.3" fill="#fff" opacity="0.8"/>`
	),
	git: badgeFileIcon('GIT', '#F05032'),
	env: badgeFileIcon('ENV', '#4CAF50'),
	lock: fileIcon(
		`<rect x="8" y="13" width="8" height="6" rx="1" fill="#78909C"/>` +
		`<path d="M10 13v-2a2 2 0 114 0v2" fill="none" stroke="#78909C" stroke-width="1.5"/>`
	),
	certificate: badgeFileIcon('CER', '#43A047'),
	key: badgeFileIcon('KEY', '#FF9800'),
	makefile: badgeFileIcon('MK', '#6D4C41'),
	shell: fileIcon(
		`<rect x="7" y="12.5" width="11" height="7.5" rx="1.2" fill="#4CAF50"/>` +
		`<path d="M9.5 16l2-1.5-2-1.5" fill="none" stroke="#fff" stroke-width="0.8"/>` +
		`<line x1="12.5" y1="16" x2="15" y2="16" stroke="#fff" stroke-width="0.8"/>`
	),
	database: fileIcon(
		`<ellipse cx="12.5" cy="14" rx="5" ry="2" fill="#0078D4"/>` +
		`<path d="M7.5 14v4c0 1.1 2.2 2 5 2s5-.9 5-2v-4" fill="none" stroke="#0078D4" stroke-width="1"/>`
	),
};

// ==================== 扩展名映射 ====================

const EXT_MAP = {
	// 编程语言
	'go': 'go',
	'js': 'javascript', 'mjs': 'javascript', 'cjs': 'javascript',
	'ts': 'typescript', 'mts': 'typescript', 'cts': 'typescript',
	'jsx': 'jsx', 'tsx': 'tsx',
	'py': 'python', 'pyw': 'python', 'pyi': 'python',
	'java': 'java', 'jar': 'java',
	'c': 'c',
	'cpp': 'cpp', 'cc': 'cpp', 'cxx': 'cpp',
	'h': 'h',
	'hpp': 'hpp', 'hh': 'hpp', 'hxx': 'hpp',
	'rs': 'rust',
	'rb': 'ruby', 'erb': 'ruby',
	'php': 'php',
	'swift': 'swift',
	'kt': 'kotlin', 'kts': 'kotlin',
	'dart': 'dart',
	'lua': 'lua',
	'pl': 'perl', 'pm': 'perl',
	'r': 'r',
	'cs': 'cs',

	// Web
	'html': 'html', 'htm': 'html',
	'css': 'css',
	'scss': 'scss',
	'sass': 'sass',
	'less': 'less',
	'vue': 'vue',
	'svelte': 'svelte',

	// 数据
	'json': 'json',
	'xml': 'xml',
	'yaml': 'yaml', 'yml': 'yaml',
	'toml': 'toml',
	'sql': 'sql',

	// 配置
	'conf': 'config', 'cfg': 'config', 'ini': 'config',
	'properties': 'config',
	'editorconfig': 'config',

	// 文档
	'txt': 'text',
	'md': 'markdown', 'mdx': 'markdown',
	'pdf': 'pdf',
	'doc': 'doc', 'docx': 'doc',
	'rtf': 'rtf',
	'log': 'log',

	// 图片
	'jpg': 'image', 'jpeg': 'image', 'png': 'image', 'gif': 'image',
	'webp': 'image', 'bmp': 'image', 'ico': 'image', 'tiff': 'image', 'tif': 'image',
	'svg': 'svg',

	// 音视频
	'mp4': 'video', 'mkv': 'video', 'avi': 'video', 'mov': 'video',
	'wmv': 'video', 'flv': 'video', 'webm': 'video',
	'mp3': 'audio', 'wav': 'audio', 'flac': 'audio', 'aac': 'audio',
	'ogg': 'audio', 'wma': 'audio', 'm4a': 'audio',

	// 压缩
	'zip': 'archive', 'rar': 'archive', '7z': 'archive',
	'tar': 'archive', 'gz': 'archive', 'tgz': 'archive', 'bz2': 'archive',
	'xz': 'archive',

	// 其他
	'env': 'env', 'lock': 'lock',
	'pem': 'certificate', 'crt': 'certificate', 'cer': 'certificate', 'key': 'key', 'pub': 'key',
	'makefile': 'makefile',
	'sh': 'shell', 'bash': 'shell', 'zsh': 'shell', 'bat': 'shell', 'ps1': 'shell',
	'db': 'database', 'sqlite': 'database',
	'gitignore': 'git', 'gitattributes': 'git',
};

// 特殊文件名映射（无扩展名或特殊名称）
const SPECIAL_FILE_MAP = {
	'dockerfile': 'docker',
	'docker-compose.yml': 'docker',
	'docker-compose.yaml': 'docker',
	'makefile': 'makefile',
	'gnumakefile': 'makefile',
	'.gitignore': 'git',
	'.gitattributes': 'git',
	'.env': 'env',
	'.env.local': 'env',
	'.env.production': 'env',
	'.env.development': 'env',
	'license': 'text',
	'license.md': 'text',
	'copying': 'text',
	'readme': 'markdown',
	'readme.md': 'markdown',
	'changelog': 'log',
	'changelog.md': 'log',
	'contributing': 'text',
	'contributing.md': 'text',
};

// ==================== 导出函数 ====================

/**
 * 获取文件图标 HTML
 * @param {string} name - 文件名
 * @param {boolean} isDir - 是否为目录
 * @returns {string} 包含图标的 span HTML
 */
export function getFileIcon(name, isDir = false) {
	if (isDir) {
		return `<span class="file-icon folder-icon" data-filename="${escapeAttr(name)}">${ICONS.folder}</span>`;
	}

	const ext = getExtension(name);
	const iconName = resolveIcon(name, ext);

	return `<span class="file-icon" data-extension="${escapeAttr(ext)}" data-filename="${escapeAttr(name)}">${ICONS[iconName] || ICONS.file}</span>`;
}

/**
 * 获取图标 CSS 类（用于颜色标记，保持向后兼容）
 * 现在统一返回空字符串，颜色已内嵌在 SVG 中
 */
export function getIconColorClass(name, isDir = false) {
	if (isDir) return 'file-icon-folder';
	return '';
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

// ==================== 内部辅助 ====================

function getExtension(filename) {
	if (!filename || !filename.includes('.')) return '';
	return filename.split('.').pop().toLowerCase();
}

function resolveIcon(filename, ext) {
	// 先检查特殊文件名
	const lowerName = filename.toLowerCase();
	if (SPECIAL_FILE_MAP[lowerName]) {
		return SPECIAL_FILE_MAP[lowerName];
	}
	// 再按扩展名查找
	if (ext && EXT_MAP[ext]) {
		return EXT_MAP[ext];
	}
	// 无扩展名的特殊文件
	if (!filename.includes('.')) {
		if (SPECIAL_FILE_MAP[lowerName]) return SPECIAL_FILE_MAP[lowerName];
		return 'file';
	}
	return 'file';
}

function escapeAttr(str) {
	return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

export default { getFileIcon, getIconColorClass, formatFileSize, formatDate };
