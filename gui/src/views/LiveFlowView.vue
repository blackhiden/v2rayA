<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { mdiClose, mdiRefresh, mdiTable, mdiVectorPolyline } from "@mdi/js";
import { useAppStore } from "@/stores/app";
import { useLiveFlowStore, type LiveFlow } from "@/stores/liveflow";
import liveFlowService from "@/plugins/liveflow";

defineOptions({ name: "LiveFlowView" });

const appStore = useAppStore();
const flowStore = useLiveFlowStore();

const viewMode = ref<"topology" | "table">("topology");

const filters = ref({
  protocol: "",
  status: "",
  proxy: "",
});

const selectedSessionId = ref<string>("");
const selectedNodeSubdomains = ref<{
  host: string;
  subdomains: { domain: string; ports: number[] }[];
} | null>(null);
const selectedNodePorts = ref<{
  host: string;
  ports: number[];
} | null>(null);

const connectionStatusText = computed(() => {
  return flowStore.connected ? "Connected" : "Disconnected";
});

const flows = computed(() => flowStore.flows);

const protocols = computed(() => {
  const set = new Set<string>();
  for (const f of flows.value) {
    if (f.protocol) set.add(f.protocol.toLowerCase());
  }
  return Array.from(set).sort();
});

const proxies = computed(() => {
  const map = new Map<string, string>();
  for (const f of flows.value) {
    for (const p of f.proxy_chain || []) {
      if (p.proxy_id && p.name) map.set(p.proxy_id, p.name);
    }
  }
  return Array.from(map.entries()).map(([id, name]) => ({ id, name }));
});

const filteredFlows = computed(() => {
  return flows.value.filter((flow) => {
    if (
      filters.value.protocol &&
      (flow.protocol || "").toLowerCase() !==
        filters.value.protocol.toLowerCase()
    ) {
      return false;
    }
    if (filters.value.status && flow.status !== filters.value.status) {
      return false;
    }
    if (filters.value.proxy) {
      const match = (flow.proxy_chain || []).some(
        (p) => p.proxy_id === filters.value.proxy,
      );
      if (!match) return false;
    }
    return true;
  });
});

const selectedFlow = computed<LiveFlow | null>(() => {
  if (!selectedSessionId.value) return null;
  return (
    filteredFlows.value.find(
      (flow) => flow.session_id === selectedSessionId.value,
    ) || null
  );
});

watch(filteredFlows, (list) => {
  if (
    selectedSessionId.value &&
    !list.some((f) => f.session_id === selectedSessionId.value)
  ) {
    selectedSessionId.value = "";
  }
});

// Graph topology calculations
const GRAPH_WIDTH = 1200;
const TOP_PADDING = 68;
const NODE_GAP = 76;
const NODE_HALF_WIDTH = 72;

const COMPOUND_TLDS = new Set([
  "co.uk",
  "co.jp",
  "co.kr",
  "co.nz",
  "co.za",
  "com.au",
  "com.br",
  "com.cn",
  "com.co",
  "com.mx",
  "com.sg",
  "com.tr",
  "com.tw",
  "com.vn",
  "net.au",
  "net.br",
  "org.au",
  "org.uk",
  "or.jp",
  "ne.jp",
  "ac.jp",
  "go.jp",
  "gob.mx",
  "edu.au",
  "gov.uk",
  "gov.sg",
]);

function rootDomain(hostname: string): string {
  if (!hostname) return hostname;
  if (/^\d{1,3}(\.\d{1,3}){3}$/.test(hostname)) return hostname;
  if (/^[\da-f]{0,4}(:[\da-f]{0,4}){2,7}$/i.test(hostname)) return hostname;

  const parts = hostname.split(".");
  if (parts.length <= 2) return hostname;

  const lastTwo = parts.slice(-2).join(".");
  if (COMPOUND_TLDS.has(lastTwo)) {
    return parts.length >= 3 ? parts.slice(-3).join(".") : hostname;
  }
  return parts.slice(-2).join(".");
}

function endpointId(endpoint: { ip: string; domain?: string }): string {
  return endpoint.domain || endpoint.ip;
}

function endpointLabel(endpoint: { ip: string; domain?: string }): string {
  return endpoint.domain || endpoint.ip;
}

function flowHops(flow: LiveFlow) {
  if (Array.isArray(flow.proxy_chain) && flow.proxy_chain.length) {
    return flow.proxy_chain;
  }
  return [
    {
      proxy_id: "route:direct",
      name: "Direct",
      type: "direct",
      server: "",
    },
  ];
}

function edgePath(
  from: { x: number; y: number },
  to: { x: number; y: number },
) {
  const startX = from.x + NODE_HALF_WIDTH;
  const endX = to.x - NODE_HALF_WIDTH;
  const curve = Math.max(36, (endX - startX) * 0.36);
  return `M ${startX} ${from.y} C ${startX + curve} ${from.y}, ${endX - curve} ${to.y}, ${endX} ${to.y}`;
}

function edgeWidth(speed?: number) {
  return !speed || speed <= 0 ? 2 : Math.min(8, 2 + Math.log10(speed + 1));
}

interface RawNode {
  id: string;
  kind: "source" | "proxy" | "destination";
  title: string;
  detail: string;
  ports: number[];
  sessionIds: string[];
  subdomains?: { domain: string; ports: number[] }[];
  subtitle?: string;
  tooltip?: string;
  x?: number;
  y?: number;
}

interface GraphNode extends RawNode {
  subtitle: string;
  tooltip: string;
  x: number;
  y: number;
}

interface GraphEdge {
  id: string;
  sessionId: string;
  status: "active" | "idle" | "error";
  width: number;
  path: string;
  label: string;
}

function addNode(
  nodes: Map<string, RawNode>,
  id: string,
  kind: "source" | "proxy" | "destination",
  title: string,
  detail: string,
  sessionId: string,
  port?: number,
) {
  const existing = nodes.get(id);
  if (existing) {
    existing.sessionIds.push(sessionId);
    if (port != null && port !== 0 && !existing.ports.includes(port)) {
      existing.ports.push(port);
      existing.ports.sort((a, b) => a - b);
    }
  } else {
    const ports = port != null && port !== 0 ? [port] : [];
    nodes.set(id, {
      id,
      kind,
      title,
      detail,
      ports,
      sessionIds: [sessionId],
    });
  }
}

function addDestinationNode(
  nodes: Map<string, RawNode>,
  id: string,
  rootDomainName: string,
  fullDomain: string,
  sessionId: string,
  port?: number,
) {
  const existing = nodes.get(id);
  if (existing) {
    existing.sessionIds.push(sessionId);
    if (port != null && port !== 0 && !existing.ports.includes(port)) {
      existing.ports.push(port);
      existing.ports.sort((a, b) => a - b);
    }
    if (existing.subdomains) {
      const sub = existing.subdomains.find((s) => s.domain === fullDomain);
      if (sub) {
        if (port != null && port !== 0 && !sub.ports.includes(port)) {
          sub.ports.push(port);
          sub.ports.sort((a, b) => a - b);
        }
      } else {
        const subPorts = port != null && port !== 0 ? [port] : [];
        existing.subdomains.push({ domain: fullDomain, ports: subPorts });
      }
    }
  } else {
    const ports = port != null && port !== 0 ? [port] : [];
    const subdomains = [
      {
        domain: fullDomain,
        ports: port != null && port !== 0 ? [port] : [],
      },
    ];
    nodes.set(id, {
      id,
      kind: "destination",
      title: rootDomainName,
      detail: "",
      ports,
      sessionIds: [sessionId],
      subdomains,
    });
  }
}

function layoutColumnNodes(nodes: RawNode[], x: number): GraphNode[] {
  return nodes.map((node, index) => ({
    ...node,
    subtitle: node.subtitle || "",
    tooltip: node.tooltip || "",
    x,
    y: TOP_PADDING + index * NODE_GAP,
  }));
}

const graph = computed(() => {
  const flowsList = filteredFlows.value;
  const maxHops = Math.max(
    1,
    ...flowsList.map((flow) => flowHops(flow).length),
  );
  const sourceX = 140;
  const destinationX = GRAPH_WIDTH - 140;
  const proxyGap = (destinationX - sourceX) / (maxHops + 1);

  const sourceMap = new Map<string, RawNode>();
  const destinationMap = new Map<string, RawNode>();
  const proxyMaps = Array.from(
    { length: maxHops },
    () => new Map<string, RawNode>(),
  );

  flowsList.forEach((flow) => {
    const sourceKey = `source:${endpointId(flow.source)}`;
    const destFullId = endpointId(flow.destination);
    const rDomain = rootDomain(destFullId);
    const destinationKey =
      destFullId === rDomain
        ? `destination:${destFullId}`
        : `destination:${rDomain}`;

    addNode(
      sourceMap,
      sourceKey,
      "source",
      endpointLabel(flow.source),
      "",
      flow.session_id,
      flow.source.port,
    );

    addDestinationNode(
      destinationMap,
      destinationKey,
      rDomain,
      destFullId,
      flow.session_id,
      flow.destination.port,
    );

    flowHops(flow).forEach((proxy, index) => {
      const proxyKey = `proxy:${index}:${proxy.proxy_id}`;
      addNode(
        proxyMaps[index],
        proxyKey,
        "proxy",
        proxy.name,
        `${proxy.type} · ${proxy.server}`,
        flow.session_id,
      );
    });
  });

  const finalizeNode = (node: RawNode): RawNode => {
    let subtitle = "";
    let tooltip = node.detail || node.id;
    if (node.kind === "source") {
      if (node.ports.length > 1) {
        subtitle = `${node.ports.length} ports`;
        tooltip = `${node.id}\nPorts: ${node.ports.join(", ")}`;
      } else if (node.ports.length === 1) {
        subtitle = `port ${node.ports[0]}`;
        tooltip = `${node.id} (port ${node.ports[0]})`;
      }
    } else if (node.kind === "destination") {
      const subCount = node.subdomains ? node.subdomains.length : 0;
      if (subCount > 1) {
        subtitle = `${subCount} subdomains`;
        const list = node
          .subdomains!.map(
            (s) =>
              `  ${s.domain}${s.ports.length ? " : " + s.ports.join(", ") : ""}`,
          )
          .join("\n");
        tooltip = `${node.title}\n${list}`;
      } else if (subCount === 1 && node.subdomains![0].domain !== node.title) {
        subtitle = node.subdomains![0].domain;
      } else if (node.ports.length > 1) {
        subtitle = `${node.ports.length} ports`;
        tooltip = `${node.id}\nPorts: ${node.ports.join(", ")}`;
      } else if (node.ports.length === 1) {
        subtitle = `port ${node.ports[0]}`;
        tooltip = `${node.id} (port ${node.ports[0]})`;
      }
    } else {
      subtitle = node.detail;
    }
    return { ...node, subtitle, tooltip };
  };

  const nodes: GraphNode[] = [
    ...layoutColumnNodes([...sourceMap.values()].map(finalizeNode), sourceX),
    ...proxyMaps.flatMap((nodeMap, index) =>
      layoutColumnNodes(
        [...nodeMap.values()].map(finalizeNode),
        sourceX + proxyGap * (index + 1),
      ),
    ),
    ...layoutColumnNodes(
      [...destinationMap.values()].map(finalizeNode),
      destinationX,
    ),
  ];

  const positions = new Map<string, GraphNode>(
    nodes.map((node) => [node.id, node]),
  );
  const edges: GraphEdge[] = [];

  flowsList.forEach((flow) => {
    const nodeIds = [`source:${endpointId(flow.source)}`];
    flowHops(flow).forEach((proxy, index) =>
      nodeIds.push(`proxy:${index}:${proxy.proxy_id}`),
    );
    const destFullId = endpointId(flow.destination);
    const rDomain = rootDomain(destFullId);
    nodeIds.push(
      destFullId === rDomain
        ? `destination:${destFullId}`
        : `destination:${rDomain}`,
    );

    nodeIds.slice(1).forEach((targetId, index) => {
      const from = positions.get(nodeIds[index]);
      const to = positions.get(targetId);
      if (!from || !to) return;
      edges.push({
        id: `${flow.session_id}:${index}`,
        sessionId: flow.session_id,
        status: flow.status || "active",
        width: edgeWidth(flow.speed_bps),
        path: edgePath(from, to),
        label: `${flow.protocol.toUpperCase()}: ${from.title} → ${to.title}`,
      });
    });
  });

  const highestNodeCount = Math.max(
    1,
    ...[sourceMap, destinationMap, ...proxyMaps].map((m) => m.size),
  );

  return {
    width: GRAPH_WIDTH,
    height: Math.max(340, TOP_PADDING + (highestNodeCount - 1) * NODE_GAP + 90),
    sourceX,
    destinationX,
    proxyColumns: proxyMaps.map((_, index) => ({
      index,
      x: sourceX + proxyGap * (index + 1),
    })),
    nodes,
    edges,
  };
});

function selectNode(node: GraphNode) {
  selectedSessionId.value = node.sessionIds[0] || "";
  if (
    node.kind === "destination" &&
    node.subdomains &&
    node.subdomains.length > 1
  ) {
    selectedNodeSubdomains.value = {
      host: node.title,
      subdomains: node.subdomains,
    };
    selectedNodePorts.value = null;
  } else if (node.ports && node.ports.length > 1) {
    selectedNodePorts.value = {
      host: node.id,
      ports: node.ports,
    };
    selectedNodeSubdomains.value = null;
  } else {
    selectedNodePorts.value = null;
    selectedNodeSubdomains.value = null;
  }
}

function selectEdge(edge: GraphEdge) {
  selectedSessionId.value = edge.sessionId;
}

function reconnect() {
  if (appStore.token) {
    liveFlowService.connect(appStore.token);
  }
}

onMounted(() => {
  if (appStore.token) {
    liveFlowService.connect(appStore.token);
  }
});

watch(
  () => appStore.token,
  (token) => {
    if (token) {
      liveFlowService.connect(token);
    } else {
      liveFlowService.disconnect();
    }
  },
);

onUnmounted(() => {
  liveFlowService.disconnect();
});
</script>

<template>
  <div class="live-flow-view">
    <div class="d-flex align-center justify-space-between mb-4 flex-wrap ga-2">
      <div>
        <h2 class="md3-title-medium ma-0">Live Flow</h2>
        <p class="md3-body-small text-on-surface-variant ma-0">
          Real-time observed routes: source → proxy → destination
        </p>
      </div>

      <div class="d-flex align-center ga-2">
        <v-btn-toggle
          v-model="viewMode"
          density="compact"
          mandatory
          variant="outlined"
          color="primary"
        >
          <v-btn value="topology" :prepend-icon="mdiVectorPolyline">
            Topology
          </v-btn>
          <v-btn value="table" :prepend-icon="mdiTable"> Table </v-btn>
        </v-btn-toggle>

        <v-chip
          size="small"
          :color="flowStore.connected ? 'success' : 'error'"
          variant="tonal"
        >
          {{ connectionStatusText }}
        </v-chip>
        <v-btn
          v-if="!flowStore.connected"
          variant="outlined"
          size="small"
          :prepend-icon="mdiRefresh"
          @click="reconnect"
        >
          Reconnect
        </v-btn>
      </div>
    </div>

    <!-- Filters and counter -->
    <v-card class="pa-3 mb-4" variant="outlined">
      <div class="d-flex align-center ga-3 flex-wrap">
        <v-select
          v-model="filters.protocol"
          label="Protocol"
          density="compact"
          hide-details
          variant="outlined"
          style="min-width: 140px; max-width: 180px"
          :items="[
            { value: '', title: 'All' },
            ...protocols.map((p) => ({ value: p, title: p.toUpperCase() })),
          ]"
        />

        <v-select
          v-model="filters.status"
          label="Status"
          density="compact"
          hide-details
          variant="outlined"
          style="min-width: 140px; max-width: 180px"
          :items="[
            { value: '', title: 'All' },
            { value: 'active', title: 'Active' },
            { value: 'idle', title: 'Idle' },
            { value: 'error', title: 'Error' },
          ]"
        />

        <v-select
          v-model="filters.proxy"
          label="Proxy"
          density="compact"
          hide-details
          variant="outlined"
          style="min-width: 180px; max-width: 240px"
          :items="[
            { value: '', title: 'All selected proxies' },
            ...proxies.map((p) => ({ value: p.id, title: p.name })),
          ]"
        />

        <v-spacer />

        <span class="md3-label-large text-on-surface-variant">
          {{ filteredFlows.length }} route{{
            filteredFlows.length === 1 ? "" : "s"
          }}
        </span>
      </div>
    </v-card>

    <!-- Graphic Visualization: SVG Topology Canvas -->
    <div v-if="viewMode === 'topology'" class="mb-4">
      <div
        class="flow-topology"
        role="region"
        aria-label="Live traffic route topology"
      >
        <div v-if="!flowStore.connected" class="flow-placeholder">
          Connecting to live route events…
        </div>
        <div v-else-if="!filteredFlows.length" class="flow-placeholder">
          Waiting for routed traffic. Start a proxied connection to see its
          path.
        </div>
        <svg
          v-else
          class="flow-svg"
          :viewBox="`0 0 ${graph.width} ${graph.height}`"
          preserveAspectRatio="xMidYMin meet"
        >
          <defs>
            <marker
              id="live-flow-arrow-active"
              markerWidth="8"
              markerHeight="8"
              refX="6"
              refY="3"
              orient="auto"
            >
              <path d="M0,0 L0,6 L7,3 z" class="flow-arrow-head active" />
            </marker>
            <marker
              id="live-flow-arrow-idle"
              markerWidth="8"
              markerHeight="8"
              refX="6"
              refY="3"
              orient="auto"
            >
              <path d="M0,0 L0,6 L7,3 z" class="flow-arrow-head idle" />
            </marker>
            <marker
              id="live-flow-arrow-error"
              markerWidth="8"
              markerHeight="8"
              refX="6"
              refY="3"
              orient="auto"
            >
              <path d="M0,0 L0,6 L7,3 z" class="flow-arrow-head error" />
            </marker>
          </defs>

          <!-- Column headers -->
          <g class="flow-columns">
            <text :x="graph.sourceX" y="28" text-anchor="middle">SOURCE</text>
            <text
              v-for="col in graph.proxyColumns"
              :key="col.index"
              :x="col.x"
              y="28"
              text-anchor="middle"
            >
              {{ col.index === 0 ? "SELECTED PROXY" : "PROXY HOP" }}
            </text>
            <text :x="graph.destinationX" y="28" text-anchor="middle">
              DESTINATION
            </text>
          </g>

          <!-- Animated Bezier directed edges -->
          <g class="flow-edges">
            <path
              v-for="edge in graph.edges"
              :key="edge.id"
              :d="edge.path"
              :class="[
                'flow-edge',
                edge.status,
                { selected: edge.sessionId === selectedSessionId },
              ]"
              :style="{ strokeWidth: edge.width }"
              :marker-end="`url(#live-flow-arrow-${edge.status})`"
              tabindex="0"
              @click="selectEdge(edge)"
              @keyup.enter="selectEdge(edge)"
            >
              <title>{{ edge.label }}</title>
            </path>
          </g>

          <!-- Topology Nodes -->
          <g class="flow-nodes">
            <g
              v-for="node in graph.nodes"
              :key="node.id"
              :class="[
                'flow-node',
                node.kind,
                { selected: node.sessionIds.includes(selectedSessionId) },
              ]"
              tabindex="0"
              @click="selectNode(node)"
              @keyup.enter="selectNode(node)"
            >
              <title>{{ node.tooltip }}</title>
              <rect
                :x="node.x - NODE_HALF_WIDTH"
                :y="node.y - 24"
                :width="NODE_HALF_WIDTH * 2"
                height="48"
                rx="10"
              />
              <text
                :x="node.x"
                :y="node.y - 5"
                text-anchor="middle"
                class="flow-node-title"
              >
                {{ node.title }}
              </text>
              <text
                :x="node.x"
                :y="node.y + 12"
                text-anchor="middle"
                class="flow-node-subtitle"
              >
                {{ node.subtitle }}
              </text>
              <text
                v-if="
                  (node.kind === 'destination' &&
                    node.subdomains &&
                    node.subdomains.length > 1) ||
                  (node.kind !== 'destination' && node.ports.length > 1)
                "
                :x="node.x + NODE_HALF_WIDTH - 12"
                :y="node.y - 14"
                class="flow-node-info"
                @click.stop="
                  node.kind === 'destination' &&
                  node.subdomains &&
                  node.subdomains.length > 1
                    ? (selectedNodeSubdomains = {
                        host: node.title,
                        subdomains: node.subdomains,
                      })
                    : (selectedNodePorts = {
                        host: node.id,
                        ports: node.ports,
                      })
                "
              >
                ⓘ
              </text>
            </g>
          </g>
        </svg>
      </div>

      <!-- Legend -->
      <div class="flow-legend pa-3 mt-2 md3-body-small">
        <span><i class="legend-line active"></i> Active route</span>
        <span><i class="legend-line idle"></i> Idle route</span>
        <span><i class="legend-line error"></i> Error route</span>
        <span><i class="legend-node proxy"></i> Selected proxy server</span>
      </div>
    </div>

    <!-- Table of active flows (accessible when in table mode) -->
    <v-card v-else variant="outlined" class="mb-4">
      <v-table hover density="comfortable">
        <thead>
          <tr>
            <th>Source</th>
            <th>Protocol</th>
            <th>Proxy Path</th>
            <th>Destination</th>
            <th class="text-center">Status</th>
            <th class="text-end">Last Activity</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="!filteredFlows.length">
            <td colspan="6" class="text-center text-on-surface-variant py-8">
              {{
                flowStore.connected
                  ? "Waiting for routed traffic. Start a proxied connection to see its path."
                  : "Connecting to live route events..."
              }}
            </td>
          </tr>
          <tr
            v-for="flow in filteredFlows"
            :key="flow.session_id"
            style="cursor: pointer"
            @click="selectedSessionId = flow.session_id"
          >
            <td class="font-weight-medium">
              {{ flow.source.ip }}:{{ flow.source.port }}
            </td>
            <td>
              <v-chip size="x-small" variant="tonal">
                {{ (flow.protocol || "tcp").toUpperCase() }}
              </v-chip>
            </td>
            <td>
              <span v-if="flow.proxy_chain && flow.proxy_chain.length">
                {{ flow.proxy_chain.map((p) => p.name).join(" → ") }}
              </span>
              <span v-else class="text-on-surface-variant">—</span>
            </td>
            <td>
              <span v-if="flow.destination.domain">
                {{ flow.destination.domain }}:{{ flow.destination.port }}
              </span>
              <span v-else>
                {{ flow.destination.ip }}:{{ flow.destination.port }}
              </span>
            </td>
            <td class="text-center">
              <v-chip
                size="x-small"
                :color="
                  flow.status === 'active'
                    ? 'success'
                    : flow.status === 'idle'
                      ? 'warning'
                      : 'error'
                "
                variant="flat"
              >
                {{ flow.status }}
              </v-chip>
            </td>
            <td class="text-end text-on-surface-variant md3-body-small">
              {{
                flow.last_activity
                  ? new Date(flow.last_activity).toLocaleTimeString()
                  : "—"
              }}
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card>

    <!-- Flow detail dialog -->
    <v-dialog
      :model-value="!!selectedFlow"
      max-width="500"
      @update:model-value="selectedSessionId = ''"
    >
      <v-card v-if="selectedFlow">
        <v-card-item class="px-6 pt-6 pb-2">
          <v-card-title class="md3-headline-small pa-0">
            Route Details
          </v-card-title>
        </v-card-item>
        <v-card-text class="px-6">
          <div class="d-flex flex-column ga-2">
            <div>
              <strong class="text-on-surface-variant md3-label-small d-block">
                SESSION ID
              </strong>
              <code>{{ selectedFlow.session_id }}</code>
            </div>
            <div>
              <strong class="text-on-surface-variant md3-label-small d-block">
                SOURCE
              </strong>
              <span
                >{{ selectedFlow.source.ip }}:{{
                  selectedFlow.source.port
                }}</span
              >
            </div>
            <div>
              <strong class="text-on-surface-variant md3-label-small d-block">
                PROTOCOL
              </strong>
              <span>{{ (selectedFlow.protocol || "tcp").toUpperCase() }}</span>
            </div>
            <div>
              <strong class="text-on-surface-variant md3-label-small d-block">
                DESTINATION
              </strong>
              <span>
                {{
                  selectedFlow.destination.domain ||
                  selectedFlow.destination.ip
                }}:{{ selectedFlow.destination.port }}
              </span>
            </div>
            <div>
              <strong class="text-on-surface-variant md3-label-small d-block">
                PROXY CHAIN
              </strong>
              <span v-if="selectedFlow.proxy_chain?.length">
                {{
                  selectedFlow.proxy_chain
                    .map((p) => `${p.name} (${p.type})`)
                    .join(" → ")
                }}
              </span>
              <span v-else>Direct</span>
            </div>
            <div>
              <strong class="text-on-surface-variant md3-label-small d-block">
                STATUS
              </strong>
              <v-chip
                size="small"
                :color="
                  selectedFlow.status === 'active' ? 'success' : 'warning'
                "
                variant="flat"
              >
                {{ selectedFlow.status }}
              </v-chip>
            </div>
          </div>
        </v-card-text>
        <v-card-actions class="px-6 pb-4">
          <v-spacer />
          <v-btn variant="text" @click="selectedSessionId = ''"> Close </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Subdomains dialog -->
    <v-dialog
      :model-value="!!selectedNodeSubdomains"
      max-width="500"
      @update:model-value="selectedNodeSubdomains = null"
    >
      <v-card v-if="selectedNodeSubdomains">
        <v-card-item class="px-6 pt-6 pb-2">
          <div class="d-flex align-center justify-space-between">
            <v-card-title class="md3-headline-small pa-0">
              Subdomains for {{ selectedNodeSubdomains.host }}
            </v-card-title>
            <v-btn
              variant="text"
              density="compact"
              :icon="mdiClose"
              @click="selectedNodeSubdomains = null"
            />
          </div>
        </v-card-item>
        <v-card-text class="px-6">
          <div class="d-flex flex-column ga-2">
            <div
              v-for="sub in selectedNodeSubdomains.subdomains"
              :key="sub.domain"
              class="d-flex align-center justify-space-between py-1 border-b"
            >
              <span class="font-monospace text-body-medium">
                {{ sub.domain }}
              </span>
              <span
                v-if="sub.ports.length"
                class="text-on-surface-variant md3-body-small"
              >
                port{{ sub.ports.length !== 1 ? "s" : "" }}:
                {{ sub.ports.join(", ") }}
              </span>
            </div>
          </div>
        </v-card-text>
        <v-card-actions class="px-6 pb-4">
          <v-spacer />
          <v-btn variant="text" @click="selectedNodeSubdomains = null">
            Close
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Ports dialog -->
    <v-dialog
      :model-value="!!selectedNodePorts"
      max-width="450"
      @update:model-value="selectedNodePorts = null"
    >
      <v-card v-if="selectedNodePorts">
        <v-card-item class="px-6 pt-6 pb-2">
          <div class="d-flex align-center justify-space-between">
            <v-card-title class="md3-headline-small pa-0">
              Active Ports
            </v-card-title>
            <v-btn
              variant="text"
              density="compact"
              :icon="mdiClose"
              @click="selectedNodePorts = null"
            />
          </div>
          <v-card-subtitle class="pa-0 mt-1">
            {{ selectedNodePorts.host }}
          </v-card-subtitle>
        </v-card-item>
        <v-card-text class="px-6">
          <div class="d-flex flex-wrap ga-2">
            <v-chip
              v-for="port in selectedNodePorts.ports"
              :key="port"
              size="small"
              variant="outlined"
            >
              {{ port }}
            </v-chip>
          </div>
        </v-card-text>
        <v-card-actions class="px-6 pb-4">
          <v-spacer />
          <v-btn variant="text" @click="selectedNodePorts = null">
            Close
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<style scoped>
.live-flow-view {
  padding-bottom: 96px;
}

.flow-topology {
  overflow-x: auto;
  min-height: 340px;
  border-radius: 12px;
  background: rgb(var(--v-theme-surface-container-low));
  border: 1px solid rgb(var(--v-theme-outline-variant));
}

.flow-svg {
  display: block;
  min-width: 800px;
  width: 100%;
}

.flow-placeholder {
  display: grid;
  min-height: 320px;
  place-items: center;
  padding: 24px;
  color: rgb(var(--v-theme-on-surface-variant));
  text-align: center;
}

.flow-columns text {
  fill: rgb(var(--v-theme-on-surface-variant));
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.08em;
}

.flow-edge {
  fill: none;
  stroke: rgb(var(--v-theme-success));
  stroke-linecap: round;
  stroke-dasharray: 8 12;
  animation: live-flow-dash 1s linear infinite;
  cursor: pointer;
  opacity: 0.72;
  transition:
    opacity 0.2s,
    stroke-width 0.2s;
}

.flow-edge.idle {
  stroke: rgb(var(--v-theme-warning));
  animation-duration: 2s;
}

.flow-edge.error {
  stroke: rgb(var(--v-theme-error));
  animation: none;
  stroke-dasharray: 3 9;
}

.flow-edge.selected,
.flow-edge:hover {
  opacity: 1;
}

.flow-arrow-head.active {
  fill: rgb(var(--v-theme-success));
}

.flow-arrow-head.idle {
  fill: rgb(var(--v-theme-warning));
}

.flow-arrow-head.error {
  fill: rgb(var(--v-theme-error));
}

@keyframes live-flow-dash {
  to {
    stroke-dashoffset: -20;
  }
}

.flow-node {
  cursor: pointer;
}

.flow-node rect {
  stroke-width: 1.5;
  transition:
    stroke-width 0.2s,
    filter 0.2s;
}

.flow-node.source rect {
  fill: rgb(var(--v-theme-primary-container));
  stroke: rgb(var(--v-theme-primary));
}

.flow-node.proxy rect {
  fill: rgb(var(--v-theme-tertiary-container));
  stroke: rgb(var(--v-theme-tertiary));
}

.flow-node.destination rect {
  fill: rgb(var(--v-theme-secondary-container));
  stroke: rgb(var(--v-theme-secondary));
}

.flow-node.selected rect,
.flow-node:hover rect {
  stroke-width: 3;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.25));
}

.flow-node-title {
  fill: rgb(var(--v-theme-on-surface));
  font-size: 12px;
  font-weight: 500;
}

.flow-node-subtitle {
  fill: rgb(var(--v-theme-on-surface-variant));
  font-size: 10px;
  font-weight: 400;
}

.flow-node-info {
  fill: rgb(var(--v-theme-primary));
  font-size: 13px;
  cursor: pointer;
}

.flow-legend {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
  color: rgb(var(--v-theme-on-surface-variant));
}

.legend-line {
  display: inline-block;
  width: 20px;
  height: 0;
  border-top: 3px dashed;
  vertical-align: middle;
}

.legend-line.active {
  border-color: rgb(var(--v-theme-success));
}

.legend-line.idle {
  border-color: rgb(var(--v-theme-warning));
}

.legend-line.error {
  border-color: rgb(var(--v-theme-error));
}

.legend-node {
  display: inline-block;
  width: 12px;
  height: 12px;
  border-radius: 3px;
  vertical-align: middle;
  border: 1.5px solid;
}

.legend-node.proxy {
  background: rgb(var(--v-theme-tertiary-container));
  border-color: rgb(var(--v-theme-tertiary));
}

@media (prefers-reduced-motion: reduce) {
  .flow-edge {
    animation: none;
  }
}
</style>
