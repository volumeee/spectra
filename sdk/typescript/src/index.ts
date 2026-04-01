import * as readline from "readline";

interface JSONRPCRequest {
  jsonrpc: string;
  id: number;
  method: string;
  params?: any;
}

interface JSONRPCResponse {
  jsonrpc: string;
  id: number;
  result?: any;
  error?: { code: number; message: string };
}

type Handler = (params: any) => Promise<any>;

export function createPlugin(name: string) {
  const handlers = new Map<string, Handler>();

  return {
    handle(method: string, handler: Handler) {
      handlers.set(method, handler);
    },

    run() {
      console.error(`[${name}] plugin started`);
      const rl = readline.createInterface({ input: process.stdin });

      rl.on("line", async (line: string) => {
        let req: JSONRPCRequest;
        try {
          req = JSON.parse(line);
        } catch {
          write({ jsonrpc: "2.0", id: 0, error: { code: -32700, message: "parse error" } });
          return;
        }

        const handler = handlers.get(req.method);
        if (!handler) {
          write({ jsonrpc: "2.0", id: req.id, error: { code: -32601, message: `method ${req.method} not found` } });
          return;
        }

        try {
          const result = await handler(req.params);
          write({ jsonrpc: "2.0", id: req.id, result });
        } catch (err: any) {
          write({ jsonrpc: "2.0", id: req.id, error: { code: -32603, message: err.message } });
        }
      });

      process.on("SIGINT", () => process.exit(0));
      process.on("SIGTERM", () => process.exit(0));
    },
  };
}

function write(resp: JSONRPCResponse) {
  process.stdout.write(JSON.stringify(resp) + "\n");
}
