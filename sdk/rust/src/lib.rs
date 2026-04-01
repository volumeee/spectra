use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::HashMap;
use std::io::{self, BufRead, Write};

#[derive(Deserialize)]
struct Request {
    jsonrpc: String,
    id: i64,
    method: String,
    params: Option<Value>,
}

#[derive(Serialize)]
struct Response {
    jsonrpc: String,
    id: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    result: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<RpcError>,
}

#[derive(Serialize)]
struct RpcError {
    code: i32,
    message: String,
}

type Handler = Box<dyn Fn(Option<Value>) -> Result<Value, String>>;

pub struct Plugin {
    name: String,
    handlers: HashMap<String, Handler>,
}

impl Plugin {
    pub fn new(name: &str) -> Self {
        Plugin {
            name: name.to_string(),
            handlers: HashMap::new(),
        }
    }

    pub fn handle<F>(&mut self, method: &str, handler: F)
    where
        F: Fn(Option<Value>) -> Result<Value, String> + 'static,
    {
        self.handlers.insert(method.to_string(), Box::new(handler));
    }

    pub fn run(&self) {
        eprintln!("[{}] plugin started", self.name);
        let stdin = io::stdin();
        let stdout = io::stdout();

        for line in stdin.lock().lines() {
            let line = match line {
                Ok(l) => l,
                Err(_) => break,
            };

            let req: Request = match serde_json::from_str(&line) {
                Ok(r) => r,
                Err(_) => {
                    write_response(&stdout, Response {
                        jsonrpc: "2.0".into(), id: 0,
                        result: None, error: Some(RpcError { code: -32700, message: "parse error".into() }),
                    });
                    continue;
                }
            };

            let handler = match self.handlers.get(&req.method) {
                Some(h) => h,
                None => {
                    write_response(&stdout, Response {
                        jsonrpc: "2.0".into(), id: req.id,
                        result: None, error: Some(RpcError { code: -32601, message: format!("method {} not found", req.method) }),
                    });
                    continue;
                }
            };

            match handler(req.params) {
                Ok(result) => write_response(&stdout, Response {
                    jsonrpc: "2.0".into(), id: req.id, result: Some(result), error: None,
                }),
                Err(msg) => write_response(&stdout, Response {
                    jsonrpc: "2.0".into(), id: req.id, result: None, error: Some(RpcError { code: -32603, message: msg }),
                }),
            }
        }
    }
}

fn write_response(stdout: &io::Stdout, resp: Response) {
    let mut out = stdout.lock();
    let _ = serde_json::to_writer(&mut out, &resp);
    let _ = out.write_all(b"\n");
    let _ = out.flush();
}
