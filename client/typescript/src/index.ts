export interface SpectraClientOptions {
  apiKey?: string;
  timeout?: number;
}

export interface ScreenshotParams {
  url: string;
  width?: number;
  height?: number;
  full_page?: boolean;
  format?: "png" | "jpeg";
  quality?: number;
}

export interface PDFParams {
  url: string;
  format?: string;
  landscape?: boolean;
}

export interface ScrapeParams {
  url: string;
  selectors?: Record<string, string>;
  wait_for?: string;
  execute_js?: string;
}

export interface APIResponse<T = any> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string };
  meta?: { request_id: string; duration_ms: number };
}

export class SpectraClient {
  private baseURL: string;
  private apiKey?: string;
  private timeout: number;

  constructor(baseURL: string, options: SpectraClientOptions = {}) {
    this.baseURL = baseURL.replace(/\/$/, "");
    this.apiKey = options.apiKey;
    this.timeout = options.timeout ?? 60000;
  }

  async screenshot(params: ScreenshotParams): Promise<any> {
    return this.execute("screenshot", "capture", params);
  }

  async pdf(params: PDFParams): Promise<any> {
    return this.execute("pdf", "generate", params);
  }

  async scrape(params: ScrapeParams): Promise<any> {
    return this.execute("scrape", "extract", params);
  }

  async plugins(): Promise<any> {
    return this.get("/api/plugins");
  }

  async health(): Promise<boolean> {
    const resp = await fetch(`${this.baseURL}/health`);
    return resp.ok;
  }

  async execute(plugin: string, method: string, params: any): Promise<any> {
    const url = `${this.baseURL}/api/${plugin}/${method}`;
    const headers: Record<string, string> = { "Content-Type": "application/json" };
    if (this.apiKey) headers["Authorization"] = `Bearer ${this.apiKey}`;

    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeout);

    try {
      const resp = await fetch(url, {
        method: "POST",
        headers,
        body: JSON.stringify(params),
        signal: controller.signal,
      });
      const json: APIResponse = await resp.json();
      if (!json.success) throw new Error(json.error?.message ?? "request failed");
      return json.data;
    } finally {
      clearTimeout(timer);
    }
  }

  private async get(path: string): Promise<any> {
    const headers: Record<string, string> = {};
    if (this.apiKey) headers["Authorization"] = `Bearer ${this.apiKey}`;
    const resp = await fetch(`${this.baseURL}${path}`, { headers });
    return resp.json();
  }
}
