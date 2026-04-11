#!/usr/bin/env node
// PixelBeast 开发模式启动脚本 (跨平台: Windows/Linux/macOS)

const { execSync, spawn } = require('child_process');
const fs = require('fs');
const os = require('os');

const platform = os.platform(); // win32 | linux | darwin
const arch = os.arch();         // x64 | arm64
const isWin = platform === 'win32';

// 设置 Go 代理
process.env.GOPROXY = 'https://goproxy.cn,direct';

// Linux 下设置 Go 环境路径
if (platform === 'linux') {
    const goroot = process.env.GOROOT || '/usr/local/go';
    const gopath = process.env.GOPATH || `${os.homedir()}/go`;
    process.env.PATH = `${goroot}/bin:${gopath}/bin:${process.env.PATH}`;
}

// ─── 平台映射 ─────────────────────────────────────────────
const goPlatform = { win32: 'windows', linux: 'linux', darwin: 'darwin' };
const goArch = { x64: 'amd64', arm64: 'arm64', ia32: '386' };

// ─── 编译 ────────────────────────────────────────────────
function build(targetPlatform, targetArch) {
    targetPlatform = targetPlatform || goPlatform[platform];
    targetArch = targetArch || goArch[arch];
    const outExt = (targetPlatform === 'windows') ? '.exe' : '';
    const outFile = `pixelbeast${outExt}`;
    const env = { ...process.env, GOOS: targetPlatform, GOARCH: targetArch };

    console.log(`\n  📦 编译 ${targetPlatform}/${targetArch} → ${outFile}\n`);

    execSync(`go build -buildvcs=false -o ${outFile}`, {
        stdio: 'inherit',
        env,
        shell: isWin,
    });

    console.log(`\n  ✅ 编译完成: ${outFile}\n`);
}

// ─── 清理 ────────────────────────────────────────────────
function clean() {
    // 先终止占用文件的进程
    try {
        if (isWin) {
            execSync('taskkill /F /IM air.exe 2>nul', { stdio: 'ignore' });
            execSync('taskkill /F /IM pixelbeast.exe 2>nul', { stdio: 'ignore' });
        } else {
            execSync('pkill -9 -f "air|pixelbeast"', { stdio: 'ignore' });
        }
    } catch {}

    const files = ['pixelbeast', 'pixelbeast.exe', 'tmp'];
    for (const f of files) {
        try { fs.rmSync(f, { recursive: true, force: true }); } catch (e) {
            console.log(`  ⚠️ 无法删除 ${f}: ${e.message}`);
        }
    }
    console.log('\n  🧹 清理完成\n');
}

// ─── 开发模式 ────────────────────────────────────────────
function dev() {
    // 清除交叉编译环境变量，确保编译当前平台
    delete process.env.GOOS;
    delete process.env.GOARCH;

    fs.mkdirSync('log', { recursive: true });
    fs.mkdirSync('tmp', { recursive: true });

    // 读取端口配置
    let adminPort = 9527;
    let httpPort = 3380;
    try {
        const cfg = JSON.parse(fs.readFileSync('config/server.json', 'utf8'));
        adminPort = cfg.admin_port || cfg.admin?.port || adminPort;
        httpPort = cfg.http_port || httpPort;
    } catch {}

    console.log();
    console.log('  🚀 PixelBeast Dev');
    console.log(`  管理面板: http://localhost:${adminPort}/admin`);
    console.log(`  默认网站: http://localhost:${httpPort}`);
    console.log('  按 Ctrl+C 停止');
    console.log();

    // 检查并安装 air
    try {
        execSync('air -v', { stdio: 'ignore', shell: isWin });
    } catch {
        console.log('安装 air...');
        execSync('go install github.com/air-verse/air@latest', { stdio: 'inherit', shell: isWin });
    }

    const air = spawn('air', [], { stdio: 'inherit', shell: isWin });

    const cleanup = () => {
        console.log('\n🛑 停止服务...');
        try {
            if (isWin) {
                execSync('taskkill /F /IM air.exe 2>nul', { stdio: 'ignore' });
                execSync('taskkill /F /IM pixelbeast.exe 2>nul', { stdio: 'ignore' });
            } else {
                execSync('pkill -9 -f "air|pixelbeast"', { stdio: 'ignore' });
            }
        } catch {}
        process.exit(0);
    };

    process.on('SIGINT', cleanup);
    process.on('SIGTERM', cleanup);
    air.on('exit', cleanup);
}

// ─── 入口 ────────────────────────────────────────────────
const cmd = process.argv[2];

switch (cmd) {
    case 'build':       build(); break;
    case 'build:win':   build('windows', 'amd64'); break;
    case 'build:linux': build('linux', 'amd64'); break;
    case 'build:arm':   build('linux', 'arm64'); break;
    case 'build:mac':   build('darwin', 'amd64'); break;
    case 'clean':       clean(); break;
    default:            dev(); break;
}
