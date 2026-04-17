/**
 * CodeMirror 6 编辑器封装
 * 本地化引入，支持搜索替换、语法高亮、行号等
 */

// CodeMirror 6 本地引入
import {
	EditorView, keymap, lineNumbers, highlightActiveLineGutter,
	highlightSpecialChars, drawSelection, dropCursor,
	rectangularSelection, crosshairCursor, highlightActiveLine,
	placeholder
} from '../vendor/codemirror.js';

import { EditorState } from '../vendor/codemirror.js';
import { defaultKeymap, history, historyKeymap, indentWithTab } from '../vendor/codemirror.js';
import { searchKeymap, highlightSelectionMatches, openSearchPanel, gotoLine } from '../vendor/codemirror.js';
import {
	syntaxHighlighting, defaultHighlightStyle, bracketMatching,
	foldGutter, indentOnInput, foldKeymap, HighlightStyle
} from '../vendor/codemirror.js';
import { closeBrackets, closeBracketsKeymap, autocompletion, completionKeymap } from '../vendor/codemirror.js';
import { lintKeymap } from '../vendor/codemirror.js';
import { tags } from '../vendor/codemirror.js';

// 语法包
import { javascript } from '../vendor/codemirror.js';
import { html } from '../vendor/codemirror.js';
import { css } from '../vendor/codemirror.js';
import { json } from '../vendor/codemirror.js';
import { xml } from '../vendor/codemirror.js';
import { python } from '../vendor/codemirror.js';
import { markdown } from '../vendor/codemirror.js';
import { sql } from '../vendor/codemirror.js';
import { cpp } from '../vendor/codemirror.js';
import { java } from '../vendor/codemirror.js';
import { rust } from '../vendor/codemirror.js';
import { php } from '../vendor/codemirror.js';
import { yaml } from '../vendor/codemirror.js';
import { lezer } from '../vendor/codemirror.js';

/**
 * VSCode Dark+ 语法配色
 */
const vscodeDarkPlusHighlight = HighlightStyle.define([
	{ tag: tags.keyword, color: '#569cd6' },
	{ tag: tags.controlKeyword, color: '#c586c0' },
	{ tag: tags.definition(tags.variableName), color: '#9cdcfe' },
	{ tag: tags.function(tags.variableName), color: '#dcdcaa' },
	{ tag: tags.variableName, color: '#9cdcfe' },
	{ tag: tags.typeName, color: '#4ec9b0' },
	{ tag: tags.propertyName, color: '#9cdcfe' },
	{ tag: tags.string, color: '#ce9178' },
	{ tag: tags.character, color: '#ce9178' },
	{ tag: tags.number, color: '#b5cea8' },
	{ tag: tags.bool, color: '#569cd6' },
	{ tag: tags.null, color: '#569cd6' },
	{ tag: tags.comment, color: '#6a9955', fontStyle: 'italic' },
	{ tag: tags.lineComment, color: '#6a9955', fontStyle: 'italic' },
	{ tag: tags.blockComment, color: '#6a9955', fontStyle: 'italic' },
	{ tag: tags.regexp, color: '#d16969' },
	{ tag: tags.escape, color: '#d7ba7d' },
	{ tag: tags.tagName, color: '#569cd6' },
	{ tag: tags.attributeName, color: '#9cdcfe' },
	{ tag: tags.attributeValue, color: '#ce9178' },
	{ tag: tags.className, color: '#4ec9b0' },
	{ tag: tags.labelName, color: '#9cdcfe' },
	{ tag: tags.operator, color: '#d4d4d4' },
	{ tag: tags.operatorKeyword, color: '#569cd6' },
	{ tag: tags.punctuation, color: '#d4d4d4' },
	{ tag: tags.bracket, color: '#ffd700' },
	{ tag: tags.atom, color: '#569cd6' },
	{ tag: tags.content, color: '#ce9178' },
	{ tag: tags.contentSeparator, color: '#6a9955' },
	{ tag: tags.list, color: '#6a9955' },
	{ tag: tags.heading, color: '#569cd6', fontWeight: 'bold' },
	{ tag: tags.strong, fontWeight: 'bold' },
	{ tag: tags.emphasis, fontStyle: 'italic' },
	{ tag: tags.link, color: '#569cd6', textDecoration: 'underline' },
	{ tag: tags.url, color: '#569cd6' },
	{ tag: tags.special(tags.string), color: '#ce9178' },
	{ tag: tags.meta, color: '#569cd6' },
	{ tag: tags.monospace, color: '#ce9178' },
	{ tag: tags.inserted, color: '#b5cea8' },
	{ tag: tags.deleted, color: '#ce9178' },
	{ tag: tags.changed, color: '#d7ba7d' },
]);

/**
 * VSCode Dark+ 编辑器外观
 */
const vscodeDarkPlusTheme = EditorView.theme({
	'&': {
		height: '100%',
		fontSize: '14px',
		backgroundColor: '#1e1e1e',
		color: '#d4d4d4',
	},
	'.cm-content': {
		fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', Consolas, monospace",
		padding: '4px 0',
		caretColor: '#aeafad',
	},
	'.cm-cursor, .cm-dropCursor': {
		borderLeftColor: '#aeafad',
		borderLeftWidth: '2px',
	},
	'&.cm-focused .cm-cursor': {
		borderLeftColor: '#aeafad',
	},
	'&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection': {
		backgroundColor: '#264f78',
	},
	'.cm-scroller': {
		fontFamily: 'inherit',
		lineHeight: '1.6',
	},
	'.cm-gutters': {
		background: '#1e1e1e',
		border: 'none',
		borderRight: '1px solid #333',
		color: '#858585',
		minWidth: '50px',
	},
	'.cm-activeLineGutter': {
		background: '#2a2d2e',
		color: '#c6c6c6',
	},
	'.cm-activeLine': {
		background: '#2a2d2e',
	},
	'.cm-matchingBracket': {
		backgroundColor: 'rgba(255, 215, 0, 0.2)',
		outline: '1px solid rgba(255, 215, 0, 0.5)',
	},
	'.cm-nonmatchingBracket': {
		backgroundColor: 'rgba(244, 67, 54, 0.2)',
		outline: '1px solid rgba(244, 67, 54, 0.5)',
	},
	'.cm-searchMatch': {
		backgroundColor: 'rgba(234, 179, 8, 0.35)',
		outline: '1px solid rgba(234, 179, 8, 0.6)',
	},
	'.cm-searchMatch.cm-searchMatch-selected': {
		backgroundColor: 'rgba(234, 179, 8, 0.6)',
	},
	'.cm-panels': {
		backgroundColor: '#252526',
		borderBottom: '1px solid #333',
		color: '#cccccc',
	},
	'.cm-panels input, .cm-panels button, .cm-panels label': {
		fontSize: '13px',
		color: '#cccccc',
	},
	'.cm-panels input': {
		backgroundColor: '#3c3c3c',
		border: '1px solid #555',
		borderRadius: '4px',
		padding: '2px 6px',
		color: '#cccccc',
	},
	'.cm-panels input:focus': {
		outline: '1px solid #007fd4',
		borderColor: '#007fd4',
	},
	'.cm-panels button': {
		backgroundColor: '#3c3c3c',
		border: '1px solid #555',
		borderRadius: '4px',
		padding: '2px 8px',
		cursor: 'pointer',
		color: '#cccccc',
	},
	'.cm-panels button:hover': {
		backgroundColor: '#505050',
	},
	'.cm-panel.cm-search': {
		padding: '4px 8px',
	},
	'.cm-foldPlaceholder': {
		backgroundColor: '#3c3c3c',
		border: 'none',
		color: '#858585',
	},
	'.cm-tooltip': {
		backgroundColor: '#252526',
		border: '1px solid #454545',
		color: '#cccccc',
	},
	'.cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]': {
		backgroundColor: '#094771',
		color: '#ffffff',
	},
	'.cm-foldGutter span': {
		color: '#858585',
	},
	'.cm-foldGutter span:hover': {
		color: '#c6c6c6',
	},
}, { dark: true });

/**
 * 搜索面板汉化短语
 */
const searchPhrases = {
	'Find': '查找',
	'Replace': '替换',
	'replace': '替换',
	'replace all': '全部替换',
	'match case': '区分大小写',
	'regexp': '正则表达式',
	'by word': '整词匹配',
	'previous': '上一个',
	'next': '下一个',
	'close': '关闭',
	'all': '全部',
	'Go to line': '跳转到行',
	'go': '跳转',
};

/**
 * 文件类型到语言支持的映射
 */
const langMap = {
	js: () => javascript({ jsx: true }),
	jsx: () => javascript({ jsx: true }),
	ts: () => javascript({ jsx: true, typescript: true }),
	tsx: () => javascript({ jsx: true, typescript: true }),
	mjs: () => javascript({ jsx: true }),
	cjs: () => javascript({ jsx: true }),
	html: () => html(),
	htm: () => html(),
	css: () => css(),
	scss: () => css(),
	less: () => css(),
	json: () => json(),
	jsonc: () => json(),
	xml: () => xml(),
	svg: () => xml(),
	py: () => python(),
	sql: () => sql(),
	c: () => cpp(),
	cpp: () => cpp(),
	h: () => cpp(),
	hpp: () => cpp(),
	java: () => java(),
	rs: () => rust(),
	php: () => php(),
	yaml: () => yaml(),
	yml: () => yaml(),
	sh: () => [],
	bash: () => [],
	md: () => markdown(),
	markdown: () => markdown(),
	go: () => [],
	ini: () => [],
	conf: () => [],
	env: () => [],
	log: () => [],
	txt: () => [],
	toml: () => [],
};

/**
 * 获取语言支持（返回数组，保证可迭代）
 */
function getLanguageSupport(ext) {
	const factory = langMap[ext];
	if (!factory) return [];
	const result = factory();
	if (Array.isArray(result)) return result;
	if (result && typeof result.extension !== 'undefined') return result.extension;
	if (result && typeof result[Symbol.iterator] === 'function') return [...result];
	return [result];
}

/**
 * 获取文件扩展名
 */
function getExtension(filename) {
	const parts = filename.split('.');
	if (parts.length > 1) {
		return parts.pop().toLowerCase();
	}
	return '';
}

/**
 * 创建 CodeMirror 编辑器
 */
export function createEditor(container, options = {}) {
	const {
		content = '',
		filename = '',
		readonly = false,
		dark = true,
		onChange,
		onSave,
	} = options;

	const ext = getExtension(filename);
	const langSupport = getLanguageSupport(ext);

	const extensions = [
		lineNumbers(),
		highlightActiveLineGutter(),
		highlightSpecialChars(),
		history(),
		foldGutter(),
		drawSelection(),
		dropCursor(),
		EditorState.allowMultipleSelections.of(true),
		indentOnInput(),
		bracketMatching(),
		closeBrackets(),
		autocompletion(),
		rectangularSelection(),
		crosshairCursor(),
		highlightActiveLine(),
		highlightSelectionMatches(),
		EditorState.phrases.of(searchPhrases),
		...langSupport,
		keymap.of([
			...closeBracketsKeymap,
			...defaultKeymap,
			...searchKeymap,
			...historyKeymap,
			...foldKeymap,
			...completionKeymap,
			...lintKeymap,
			indentWithTab,
			// VS Code 风格：Ctrl+G 跳转到行
			{ key: 'Mod-g', run: gotoLine },
		]),
	];

	if (dark) {
		extensions.push(vscodeDarkPlusTheme);
		extensions.push(syntaxHighlighting(vscodeDarkPlusHighlight, { fallback: true }));
	} else {
		extensions.push(syntaxHighlighting(defaultHighlightStyle, { fallback: true }));
	}

	if (readonly) {
		extensions.push(EditorState.readOnly.of(true));
	}

	if (onChange) {
		extensions.push(EditorView.updateListener.of((update) => {
			if (update.docChanged) {
				onChange(update.state.doc.toString());
			}
		}));
	}

	if (onSave) {
		extensions.push(keymap.of([{
			key: 'Mod-s',
			run: () => {
				onSave();
				return true;
			}
		}]));
	}

	const state = EditorState.create({
		doc: content,
		extensions,
	});

	const view = new EditorView({
		state,
		parent: container,
	});

	return view;
}

export function getContent(view) {
	return view.state.doc.toString();
}

export function setContent(view, content) {
	view.dispatch({
		changes: { from: 0, to: view.state.doc.length, insert: content },
	});
}

export function focusEditor(view) {
	view.focus();
}

export function openSearch(view) {
	openSearchPanel(view);
}

export function destroyEditor(view) {
	view.destroy();
}
