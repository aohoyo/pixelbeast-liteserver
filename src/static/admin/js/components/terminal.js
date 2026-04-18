/**
 * Web 终端组件
 * 基于 xterm.js + WebSocket + PTY
 */
import { Terminal } from '../vendor/xterm.js';
import { FitAddon } from '../vendor/xterm.js';
import { WebLinksAddon } from '../vendor/xterm.js';

export class WebTerminal {
    constructor(container, options = {}) {
        this.container = container;
        this.options = options;
        this.ws = null;
        this.term = null;
        this.fitAddon = null;
        this.connected = false;
        this.initCommand = options.initCommand || '';
    }

    async init() {
        // 创建 xterm 实例
        this.term = new Terminal({
            cursorBlink: true,
            fontSize: 14,
            fontFamily: "'Cascadia Code', 'Fira Code', 'JetBrains Mono', 'Consolas', monospace",
            theme: {
                background: '#1a1a2e',
                foreground: '#d4d4d4',
                cursor: '#d4d4d4',
                selectionBackground: '#264f78',
                black: '#000000',
                red: '#cd3131',
                green: '#0dbc79',
                yellow: '#e5e510',
                blue: '#2472c8',
                magenta: '#bc3fbc',
                cyan: '#11a8cd',
                white: '#e5e5e5',
                brightBlack: '#666666',
                brightRed: '#f14c4c',
                brightGreen: '#23d18b',
                brightYellow: '#f5f543',
                brightBlue: '#3b8eea',
                brightMagenta: '#d670d6',
                brightCyan: '#29b8db',
                brightWhite: '#ffffff'
            }
        });

        this.fitAddon = new FitAddon();
        this.term.loadAddon(this.fitAddon);
        this.term.loadAddon(new WebLinksAddon());

        this.term.open(this.container);
        this.fitAddon.fit();

        // 连接 WebSocket
        this.connect();

        // 监听输入
        this.term.onData(data => {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                this.ws.send(JSON.stringify({ type: 'input', data }));
            }
        });

        // 窗口 resize
        this._resizeHandler = () => {
            if (this.fitAddon) {
                this.fitAddon.fit();
                this.sendResize();
            }
        };
        window.addEventListener('resize', this._resizeHandler);
    }

    connect() {
        const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        let url = `${proto}//${location.host}/api/terminal/ws`;
        if (this.options.cwd) {
            url += `?cwd=${encodeURIComponent(this.options.cwd)}`;
        }

        this.term.writeln('\x1b[90m正在连接终端...\x1b[0m');

        this.ws = new WebSocket(url);

        this.ws.onopen = () => {
            this.connected = true;
            this.sendResize();
            // 发送初始命令
            if (this.initCommand) {
                this.ws.send(JSON.stringify({ type: 'input', data: this.initCommand + '\n' }));
                this.initCommand = '';
            }
        };

        this.ws.onmessage = (event) => {
            this.term.write(event.data);
        };

        this.ws.onclose = () => {
            this.connected = false;
            this.term.writeln('\x1b[90m\r\n连接已断开\x1b[0m');
        };

        this.ws.onerror = () => {
            this.connected = false;
            this.term.writeln('\x1b[31m\r\n连接失败\x1b[0m');
        };
    }

    sendResize() {
        if (this.ws && this.ws.readyState === WebSocket.OPEN && this.term) {
            this.ws.send(JSON.stringify({
                type: 'resize',
                rows: this.term.rows,
                cols: this.term.cols
            }));
        }
    }

    runCommand(cmd) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type: 'input', data: cmd + '\n' }));
        }
    }

    destroy() {
        window.removeEventListener('resize', this._resizeHandler);
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        if (this.term) {
            this.term.dispose();
            this.term = null;
        }
    }
}
