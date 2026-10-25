import { fileURLToPath, URL } from "node:url";
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import net from "node:net";

// 探测一个可用端口：跳过被系统保留（EACCES）或已被占用（EADDRINUSE）的端口，
// 彻底规避 Windows 保留端口段（Hyper-V / WSL / WinNAT）导致的 "listen EACCES" 崩溃。
function findAvailablePort(start: number, maxTries = 200): Promise<number> {
  return new Promise((resolve, reject) => {
    const attempt = (port: number) => {
      if (port - start >= maxTries) {
        reject(new Error("在尝试范围内未找到可用端口"));
        return;
      }
      const server = net.createServer();
      server.once("error", (err: NodeJS.ErrnoException) => {
        // EACCES = 端口被系统保留段占用；EADDRINUSE = 端口已被其他进程占用。两者都顺延。
        if (err.code === "EACCES" || err.code === "EADDRINUSE") {
          server.close(() => attempt(port + 1));
        } else {
          reject(err);
        }
      });
      server.listen(port, probeHost, () => {
        const addr = server.address() as net.AddressInfo;
        server.close(() => resolve(addr.port));
      });
    };
    attempt(start);
  });
}

// 是否通过 `npm run dev -- --host` 要求局域网/其他设备访问。
// 该情况下通配绑定 0.0.0.0 也可能命中保留段，因此探测也要用 0.0.0.0。
const wantsLan = process.argv.includes("--host");
const probeHost = wantsLan ? "0.0.0.0" : "127.0.0.1";

export default defineConfig(async () => {
  const port = await findAvailablePort(5173);
  return {
    plugins: [vue()],
    resolve: {
      alias: {
        "@": fileURLToPath(new URL("./src", import.meta.url)),
      },
    },
    server: {
      // 默认仅绑定回环地址（localhost），避免通配绑定 0.0.0.0 触发的 EACCES。
      // 需要局域网访问时：npm run dev -- --host
      host: wantsLan ? true : "localhost",
      port,
      // 端口已由上面的探测确认可用，固定使用，避免 Vite 再次随机顺延引发困惑。
      strictPort: true,
      proxy: {
        "/api": {
          target: "http://localhost:9080",
          changeOrigin: true,
        },
      },
    },
  };
});
