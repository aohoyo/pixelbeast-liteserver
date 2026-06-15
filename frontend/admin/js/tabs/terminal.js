/**
 * 终端 Tab 模块
 */
import { WebTerminal } from '../components/terminal.js';
import { globalEvents } from '../core/events.js';

let terminal = null;
let initialized = false;

export function initTerminalTab(deps) {
    if (initialized) return;

    const container = document.getElementById('terminal-container');
    if (!container) return;

    // 加载 xterm CSS
    if (!document.querySelector('link[href*="xterm.css"]')) {
        const link = document.createElement('link');
        link.rel = 'stylesheet';
        link.href = 'css/xterm.css';
        document.head.appendChild(link);
    }

    terminal = new WebTerminal(container, deps);
    terminal.init();
    initialized = true;

    // 暴露到全局，供文件管理器调用
    window.__pixelbeast_terminal = terminal;
}

// 监听 Tab 切换，终端需要 resize
globalEvents.match('tab:switch:*', () => {
    if (terminal && terminal.fitAddon) {
        setTimeout(() => terminal.fitAddon.fit(), 100);
    }
});
