import { defineStore } from "pinia";

export interface FlowEndpoint {
  ip: string;
  port: number;
  domain: string;
}

export interface FlowProxy {
  proxy_id: string;
  name: string;
  type: string;
  server: string;
}

export interface LiveFlow {
  session_id: string;
  source: FlowEndpoint;
  proxy_chain: FlowProxy[];
  destination: FlowEndpoint;
  protocol: string;
  start_time: string;
  bytes_sent: number;
  bytes_recv: number;
  speed_bps: number;
  last_activity: string;
  status: "active" | "idle" | "error";
}

export interface LiveFlowWarning {
  message: string;
}

export const useLiveFlowStore = defineStore("liveflow", {
  state: () => ({
    flows: [] as LiveFlow[],
    connected: false,
    warnings: [] as LiveFlowWarning[],
  }),
  actions: {
    setConnected(val: boolean) {
      this.connected = val;
    },
    setFlows(flows: LiveFlow[]) {
      this.flows = flows;
    },
    addFlow(flow: LiveFlow) {
      const idx = this.flows.findIndex((f) => f.session_id === flow.session_id);
      if (idx >= 0) {
        this.flows[idx] = flow;
      } else {
        this.flows.push(flow);
      }
    },
    updateFlow(update: Partial<LiveFlow> & { session_id: string }) {
      const flow = this.flows.find((f) => f.session_id === update.session_id);
      if (flow) {
        Object.assign(flow, update);
      }
    },
    removeFlow(sessionId: string) {
      this.flows = this.flows.filter((f) => f.session_id !== sessionId);
    },
    setWarnings(warnings: LiveFlowWarning[]) {
      this.warnings = warnings;
    },
    clearWarnings() {
      this.warnings = [];
    },
  },
});
