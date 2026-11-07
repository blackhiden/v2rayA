import { apiRoot } from "@/api/client";
import {
  useLiveFlowStore,
  type FlowEndpoint,
  type FlowProxy,
  type LiveFlow,
} from "@/stores/liveflow";

function isObject(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function normalizeEndpoint(value: unknown): FlowEndpoint | null {
  if (!isObject(value) || typeof value.ip !== "string" || !value.ip) {
    return null;
  }
  return {
    ip: value.ip,
    port: typeof value.port === "number" ? value.port : 0,
    domain: typeof value.domain === "string" ? value.domain : "",
  };
}

function normalizeProxyChain(value: unknown): FlowProxy[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value
    .filter(
      (proxy): proxy is Record<string, unknown> =>
        isObject(proxy) &&
        typeof proxy.proxy_id === "string" &&
        typeof proxy.name === "string",
    )
    .map((proxy) => ({
      proxy_id: proxy.proxy_id as string,
      name: proxy.name as string,
      type: typeof proxy.type === "string" ? proxy.type : "unknown",
      server: typeof proxy.server === "string" ? proxy.server : "",
    }));
}

function normalizeFlowStart(data: unknown): LiveFlow | null {
  if (
    !isObject(data) ||
    typeof data.session_id !== "string" ||
    !data.session_id
  ) {
    return null;
  }

  const source = normalizeEndpoint(data.source);
  const destination = normalizeEndpoint(data.destination);
  if (!source || !destination || typeof data.protocol !== "string") {
    return null;
  }

  return {
    session_id: data.session_id,
    source,
    proxy_chain: normalizeProxyChain(data.proxy_chain),
    destination,
    protocol: data.protocol || "tcp",
    start_time: (data.start_time as string) || new Date().toISOString(),
    bytes_sent: 0,
    bytes_recv: 0,
    speed_bps: 0,
    last_activity: (data.start_time as string) || new Date().toISOString(),
    status: "active",
  };
}

class LiveFlowService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;
  private reconnectDelay = 1000;
  private shouldReconnect = false;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  buildUrl(token: string): string {
    const root = apiRoot();
    try {
      const url = new URL(root, window.location.href);
      const protocol = url.protocol === "https:" ? "wss:" : "ws:";
      const basePath = url.pathname.replace(/\/api\/?$/, "");
      return `${protocol}//${url.host}${basePath}/api/live-flow?token=${encodeURIComponent(token)}`;
    } catch {
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      return `${protocol}//${window.location.host}/api/live-flow?token=${encodeURIComponent(token)}`;
    }
  }

  connect(token: string) {
    if (!token) return;
    if (
      this.ws &&
      (this.ws.readyState === WebSocket.OPEN ||
        this.ws.readyState === WebSocket.CONNECTING)
    ) {
      return;
    }

    this.shouldReconnect = true;
    const store = useLiveFlowStore();

    try {
      this.ws = new WebSocket(this.buildUrl(token));

      this.ws.onopen = () => {
        this.reconnectAttempts = 0;
        store.setConnected(true);
      };

      this.ws.onmessage = (event) => {
        try {
          this.handleMessage(JSON.parse(event.data));
        } catch (error) {
          console.warn("[LiveFlow] Ignored malformed WebSocket message", error);
        }
      };

      this.ws.onclose = () => {
        this.ws = null;
        store.setConnected(false);
        this.scheduleReconnect(token);
      };

      this.ws.onerror = () => {
        store.setConnected(false);
      };
    } catch (err) {
      console.warn("[LiveFlow] Connection failed", err);
    }
  }

  scheduleReconnect(token: string) {
    if (
      !this.shouldReconnect ||
      this.reconnectAttempts >= this.maxReconnectAttempts
    ) {
      return;
    }

    const delay = Math.min(
      this.reconnectDelay * Math.pow(2, this.reconnectAttempts),
      30000,
    );
    this.reconnectAttempts += 1;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.reconnectTimer = setTimeout(() => this.connect(token), delay);
  }

  handleMessage(message: unknown) {
    if (!isObject(message) || typeof message.type !== "string") {
      return;
    }

    const store = useLiveFlowStore();

    switch (message.type) {
      case "flow_start": {
        const flow = normalizeFlowStart(message.data);
        if (flow) {
          store.addFlow(flow);
        }
        break;
      }
      case "flow_update":
        if (
          isObject(message.data) &&
          typeof message.data.session_id === "string"
        ) {
          store.updateFlow(
            message.data as Partial<LiveFlow> & { session_id: string },
          );
        }
        break;
      case "flow_end":
        if (
          isObject(message.data) &&
          typeof message.data.session_id === "string"
        ) {
          store.removeFlow(message.data.session_id as string);
        }
        break;
      case "batch_state": {
        const sessions =
          isObject(message.data) && Array.isArray(message.data.sessions)
            ? (message.data.sessions.map(normalizeFlowStart).filter(Boolean) as LiveFlow[])
            : [];
        store.setFlows(sessions);
        break;
      }
      case "error":
        if (
          isObject(message.data) &&
          typeof message.data.message === "string"
        ) {
          store.setWarnings([{ message: message.data.message as string }]);
        }
        break;
    }
  }

  disconnect() {
    this.shouldReconnect = false;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    const store = useLiveFlowStore();
    store.setConnected(false);
  }
}

export default new LiveFlowService();
