import http from "node:http";
import fs from "node:fs/promises";
import path from "node:path";
import crypto from "node:crypto";
import net from "node:net";

const root = path.resolve(process.argv[2] || new URL("./static", import.meta.url).pathname);
const port = Number(process.env.PORT || 8765);

const types = new Map([
  [".html", "text/html; charset=utf-8"],
  [".js", "text/javascript; charset=utf-8"],
  [".wasm", "application/wasm"],
]);

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, `http://${req.headers.host}`);
  const pathname = url.pathname === "/" ? "/index.html" : url.pathname;
  const filePath = path.resolve(path.join(root, path.normalize(pathname)));

  if (!filePath.startsWith(root)) {
    res.writeHead(403);
    res.end("forbidden");
    return;
  }

  try {
    const body = await fs.readFile(filePath);
    res.writeHead(200, {
      "content-type": types.get(path.extname(filePath)) || "application/octet-stream",
      "cross-origin-opener-policy": "same-origin",
      "cross-origin-embedder-policy": "require-corp",
      "cross-origin-resource-policy": "same-origin",
    });
    res.end(body);
  } catch (error) {
    res.writeHead(404);
    res.end("not found");
  }
});

server.on("upgrade", (req, socket) => {
  const url = new URL(req.url, `http://${req.headers.host}`);
  if (url.pathname !== "/peer-proxy") {
    socket.destroy();
    return;
  }

  const target = url.searchParams.get("target");
  if (!target) {
    socket.destroy();
    return;
  }

  const [host, portString] = target.split(":");
  const port = Number(portString);
  if (host !== "127.0.0.1" || !Number.isInteger(port)) {
    socket.destroy();
    return;
  }

  const key = req.headers["sec-websocket-key"];
  if (!key) {
    socket.destroy();
    return;
  }

  const accept = crypto
    .createHash("sha1")
    .update(`${key}258EAFA5-E914-47DA-95CA-C5AB0DC85B11`)
    .digest("base64");

  socket.write([
    "HTTP/1.1 101 Switching Protocols",
    "Upgrade: websocket",
    "Connection: Upgrade",
    `Sec-WebSocket-Accept: ${accept}`,
    "",
    "",
  ].join("\r\n"));

  const tcp = net.connect({ host, port });
  let buffer = Buffer.alloc(0);

  socket.on("data", (chunk) => {
    buffer = Buffer.concat([buffer, chunk]);
    while (true) {
      const frame = readFrame(buffer);
      if (!frame) {
        break;
      }

      buffer = buffer.subarray(frame.consumed);
      if (frame.opcode === 0x8) {
        socket.end();
        tcp.end();
        return;
      }
      if (frame.opcode === 0x2 || frame.opcode === 0x0) {
        tcp.write(frame.payload);
      }
    }
  });

  tcp.on("data", (chunk) => socket.write(writeBinaryFrame(chunk)));
  tcp.on("error", () => socket.destroy());
  tcp.on("close", () => socket.end());
  socket.on("error", () => tcp.destroy());
  socket.on("close", () => tcp.destroy());
});

server.listen(port, "127.0.0.1", () => {
  console.log(`lnd wasm demo: http://127.0.0.1:${port}`);
});

function readFrame(buffer) {
  if (buffer.length < 2) {
    return null;
  }

  const opcode = buffer[0] & 0x0f;
  const masked = (buffer[1] & 0x80) !== 0;
  let length = buffer[1] & 0x7f;
  let offset = 2;

  if (length === 126) {
    if (buffer.length < offset + 2) {
      return null;
    }
    length = buffer.readUInt16BE(offset);
    offset += 2;
  } else if (length === 127) {
    if (buffer.length < offset + 8) {
      return null;
    }
    const high = buffer.readUInt32BE(offset);
    const low = buffer.readUInt32BE(offset + 4);
    length = high * 2 ** 32 + low;
    offset += 8;
  }

  if (!masked || buffer.length < offset + 4 + length) {
    return null;
  }

  const mask = buffer.subarray(offset, offset + 4);
  offset += 4;

  const payload = Buffer.from(buffer.subarray(offset, offset + length));
  for (let i = 0; i < payload.length; i += 1) {
    payload[i] ^= mask[i % 4];
  }

  return {
    opcode,
    payload,
    consumed: offset + length,
  };
}

function writeBinaryFrame(payload) {
  const length = payload.length;
  if (length < 126) {
    return Buffer.concat([Buffer.from([0x82, length]), payload]);
  }
  if (length <= 0xffff) {
    const header = Buffer.alloc(4);
    header[0] = 0x82;
    header[1] = 126;
    header.writeUInt16BE(length, 2);
    return Buffer.concat([header, payload]);
  }

  const header = Buffer.alloc(10);
  header[0] = 0x82;
  header[1] = 127;
  header.writeBigUInt64BE(BigInt(length), 2);
  return Buffer.concat([header, payload]);
}
