#!/usr/bin/env node
// PixelBeast LiteServer - 开发 & 构建脚本 (Windows/Linux/macOS)

const { execSync, spawn } = require('child_process');
const fs = require('fs');
const path = require('path');
const os = require('os');

// ─── 全局配置 ──────────────────────────────────────────────
const PLATFORM = os.platform();  // win32 | linux | darwin
const ARCH = os.arch();          // x64 | arm64 | ia32
const IS_WIN = PLATFORM === 'win32';
const ROOT = path.resolve(__dirname);
const ENTRY = './src/cmd';       // Go 入口包
const VERSION = require('./package.json').version;

// Go 代理
process.env.GOPROXY = process.env.GOPROXY || 'https://goproxy.cn,direct';

// Linux/macOS 确保 Go 在 PATH 中
if (PLATFORM !== 'win32') {
    const goroot = process.env.GOROOT || '/usr/local/go';
    const gopath = process.env.GOPATH || `${os.homedir()}/go`;
    const paths = [`${goroot}/bin`, `${gopath}/bin`];
    process.env.PATH = [...paths, ...process.env.PATH.split(path.delimiter)]
        .filter((v, i, a) => a.indexOf(v) === i)
        .join(path.delimiter);
}

// 平台映射
const GO_OS = { win32: 'windows', linux: 'linux', darwin: 'darwin' };
const GO_ARCH = { x64: 'amd64', arm64: 'arm64', ia32: '386' };

// ─── 工具函数 ──────────────────────────────────────────────
function run(cmd, opts = {}) {
    return execSync(cmd, {
        cwd: ROOT,
        stdio: 'inherit',
        shell: IS_WIN || opts.shell,
        env: { ...process.env, ...opts.env },
    });
}

function runQuiet(cmd) {
    try {
        return execSync(cmd, { cwd: ROOT, encoding: 'utf8', shell: IS_WIN }).trim();
    } catch {
        return '';
    }
}

function getBinaryName(targetOS) {
    return (targetOS || GO_OS[PLATFORM]) === 'windows' ? 'pixelbeast.exe' : 'pixelbeast';
}

function getAdminPort() {
    try {
        const cfg = JSON.parse(fs.readFileSync(path.join(ROOT, 'config/server.json'), 'utf8'));
        return cfg.admin?.port || cfg.admin_port || 9527;
    } catch {
        return 9527;
    }
}

function ensureDir(dir) {
    const p = path.join(ROOT, dir);
    if (!fs.existsSync(p)) fs.mkdirSync(p, { recursive: true });
}

function killProcess(name) {
    try {
        if (IS_WIN) {
            execSync(`taskkill /F /IM ${name} 2>nul`, { stdio: 'ignore', shell: true });
        } else {
            execSync(`pkill -9 -f "${name}"`, { stdio: 'ignore' });
        }
    } catch {}
}

function printBanner(mode) {
    const port = getAdminPort();
    console.log();
    console.log(`  \x1b[35mPixelBeast LiteServer v${VERSION}\x1b[0m`);
    console.log(`  \x1b[36m模式: ${mode}\x1b[0m`);
    console.log(`  \x1b[36m平台: ${PLATFORM}/${ARCH}\x1b[0m`);
    console.log();
    console.log(`  管理面板: \x1b[32mhttp://localhost:${port}/admin\x1b[0m`);
    console.log(`  按 Ctrl+C 停止`);
    console.log();
}

// ─── 编译 ──────────────────────────────────────────────────
function build(targetOS, targetArch) {
    targetOS = targetOS || GO_OS[PLATFORM];
    targetArch = targetArch || GO_ARCH[ARCH];
    const outFile = getBinaryName(targetOS);
    const env = { GOOS: targetOS, GOARCH: targetArch };

    // 从 go.mod 读取版本或用 package.json 版本
    const buildTime = new Date().toISOString().replace(/\.\d+Z$/, 'Z');

    console.log();
    console.log(`  \x1b[33m编译 ${targetOS}/${targetArch} → ${outFile}\x1b[0m`);
    console.log();

    const ldflags = `-s -w -X main.version=${VERSION} -X main.buildTime=${buildTime}`;
    run(`go build -buildvcs=false -ldflags "${ldflags}" -o ${outFile} ${ENTRY}`, { env });

    const stat = fs.statSync(path.join(ROOT, outFile));
    const sizeMB = (stat.size / 1024 / 1024).toFixed(1);
    console.log();
    console.log(`  \x1b[32m编译完成: ${outFile} (${sizeMB} MB)\x1b[0m`);
    console.log();
}

// ─── 开发模式（go run，无需 air）──────────────────────────
function devGoRun() {
    printBanner('go run（直接运行）');
    ensureDir('log');
    ensureDir('tmp');

    // 清除交叉编译环境变量
    delete process.env.GOOS;
    delete process.env.GOARCH;

    const cleanup = () => {
        console.log('\n  \x1b[33m停止服务...\x1b[0m');
        killProcess('pixelbeast');
        process.exit(0);
    };

    const proc = spawn('go', ['run', ENTRY, '-config', 'config'], {
        cwd: ROOT,
        stdio: 'inherit',
        shell: IS_WIN,
        env: process.env,
    });

    process.on('SIGINT', cleanup);
    process.on('SIGTERM', cleanup);
    proc.on('exit', () => cleanup());
}

// ─── 开发模式（air 热重载）─────────────────────────────────
function devAir() {
    printBanner('air（热重载）');
    ensureDir('log');
    ensureDir('tmp');

    // 清除交叉编译环境变量
    delete process.env.GOOS;
    delete process.env.GOARCH;

    // 生成 .air.toml
    generateAirConfig();

    // 检查并安装 air
    try {
        runQuiet('air -v');
    } catch {
        console.log('  \x1b[33m安装 air...\x1b[0m');
        run('go install github.com/air-verse/air@latest');
    }

    const cleanup = () => {
        console.log('\n  \x1b[33m停止服务...\x1b[0m');
        killProcess('air');
        killProcess('pixelbeast');
        process.exit(0);
    };

    const proc = spawn('air', [], {
        cwd: ROOT,
        stdio: 'inherit',
        shell: IS_WIN,
        env: process.env,
    });

    process.on('SIGINT', cleanup);
    process.on('SIGTERM', cleanup);
    proc.on('exit', () => cleanup());
}

function generateAirConfig() {
    const binName = getBinaryName();
    const config = `# PixelBeast LiteServer - Air 热重载配置（自动生成）

root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -buildvcs=false -o ./tmp/${binName} ${ENTRY}"
  bin = "./tmp/${binName}"
  args = ["-config", "config"]
  include_ext = ["go", "html", "css", "js"]
  exclude_ext = ["_test.go"]
  include_dir = ["src"]
  exclude_dir = ["tmp", "vendor", "testdata", "sites", "ftp", "log", "docs", "node_modules", "config", "ssl", "backups"]
  delay = 1000
  stop_on_root = true
  send_interrupt = true
  kill_delay = 500

[log]
  time = true
  main_only = false

[color]
  main = "magenta"
  watcher = "cyan"
  build = "yellow"
  runner = "green"

[misc]
  clean_on_exit = true

[screen]
  clear_on_rebuild = false
  keep_scroll = true
`;
    const airPath = path.join(ROOT, '.air.toml');
    // 只在文件不存在或内容变化时写入，避免触发 air 重载循环
    const existing = fs.existsSync(airPath) ? fs.readFileSync(airPath, 'utf8') : '';
    if (existing !== config) {
        fs.writeFileSync(airPath, config, 'utf8');
        console.log('  \x1b[36m已生成 .air.toml\x1b[0m');
    }
}

// ─── 清理 ──────────────────────────────────────────────────
function clean() {
    killProcess('air');
    killProcess('pixelbeast.exe');
    killProcess('pixelbeast');

    const files = [
        'pixelbeast', 'pixelbeast.exe',
        path.join('tmp', 'pixelbeast'), path.join('tmp', 'pixelbeast.exe'),
        'tmp',
    ];
    for (const f of files) {
        try {
            fs.rmSync(path.join(ROOT, f), { recursive: true, force: true });
        } catch {}
    }
    console.log('\n  \x1b[32m清理完成\x1b[0m\n');
}

// ─── 版本 ──────────────────────────────────────────────────
function showVersion() {
    console.log(`PixelBeast LiteServer v${VERSION}`);
    const goVer = runQuiet('go version');
    if (goVer) console.log(goVer);
}

// ─── 日志清理 ──────────────────────────────────────────────
function clearLogs() {
    const logDir = path.join(ROOT, 'log');
    try {
        fs.rmSync(logDir, { recursive: true, force: true });
    } catch {}
    fs.mkdirSync(logDir, { recursive: true });
    console.log('\n  \x1b[32m日志已清理\x1b[0m\n');
}

// ─── 入口 ──────────────────────────────────────────────────
const cmd = process.argv[2] || 'dev';

switch (cmd) {
    case 'dev':        devGoRun(); break;
    case 'dev:air':    devAir(); break;
    case 'build':      build(); break;
    case 'build:win':  build('windows', 'amd64'); break;
    case 'build:linux': build('linux', 'amd64'); break;
    case 'build:arm':  build('linux', 'arm64'); break;
    case 'build:mac':  build('darwin', 'amd64'); break;
    case 'build:mac-arm': build('darwin', 'arm64'); break;
    case 'clean':      clean(); break;
    case 'version':    showVersion(); break;
    case 'logs:clear': clearLogs(); break;
    default:
        console.log(`\n  未知命令: ${cmd}\n`);
        console.log('  可用命令:');
        console.log('    dev          开发模式（go run）');
        console.log('    dev:air      开发模式（air 热重载）');
        console.log('    build        编译当前平台');
        console.log('    build:win    交叉编译 Windows amd64');
        console.log('    build:linux  交叉编译 Linux amd64');
        console.log('    build:arm    交叉编译 Linux arm64');
        console.log('    build:mac    交叉编译 macOS amd64');
        console.log('    build:mac-arm 交叉编译 macOS arm64');
        console.log('    clean        清理编译产物');
        console.log('    version      显示版本信息');
        console.log('    logs:clear   清理日志');
        console.log();
        process.exit(1);
}
