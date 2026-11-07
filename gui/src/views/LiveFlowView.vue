<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { mdiRefresh } from "@mdi/js";
import { useAppStore } from "@/stores/app";
import { useLiveFlowStore, type LiveFlow } from "@/stores/liveflow";
import liveFlowService from "@/plugins/liveflow";

defineOptions({ name: "LiveFlowView" });

const appStore = useAppStore();
const flowStore = useLiveFlowStore();

const filters = ref({
  protocol: "",
  status: "",
  proxy: "",
});

const selectedFlow = ref<LiveFlow | null>(null);

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
    if (filters.value.protocol && (flow.protocol || "").toLowerCase() !== filters.value.protocol.toLowerCase()) {
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
          style="min-width: 140px; max-width: 180px;"
          :items="[{ value: '', title: 'All' }, ...protocols.map(p => ({ value: p, title: p.toUpperCase() }))]"
        />

        <v-select
          v-model="filters.status"
          label="Status"
          density="compact"
          hide-details
          variant="outlined"
          style="min-width: 140px; max-width: 180px;"
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
          style="min-width: 180px; max-width: 240px;"
          :items="[{ value: '', title: 'All selected proxies' }, ...proxies.map(p => ({ value: p.id, title: p.name }))]"
        />

        <v-spacer />

        <span class="md3-label-large text-on-surface-variant">
          {{ filteredFlows.length }} route{{ filteredFlows.length === 1 ? '' : 's' }}
        </span>
      </div>
    </v-card>

    <!-- Table of active flows -->
    <v-card variant="outlined">
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
              {{ flowStore.connected ? 'Waiting for routed traffic. Start a proxied connection to see its path.' : 'Connecting to live route events...' }}
            </td>
          </tr>
          <tr
            v-for="flow in filteredFlows"
            :key="flow.session_id"
            style="cursor: pointer;"
            @click="selectedFlow = flow"
          >
            <td class="font-weight-medium">
              {{ flow.source.ip }}:{{ flow.source.port }}
            </td>
            <td>
              <v-chip size="x-small" variant="tonal">
                {{ (flow.protocol || 'tcp').toUpperCase() }}
              </v-chip>
            </td>
            <td>
              <span v-if="flow.proxy_chain && flow.proxy_chain.length">
                {{ flow.proxy_chain.map(p => p.name).join(' → ') }}
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
                :color="flow.status === 'active' ? 'success' : flow.status === 'idle' ? 'warning' : 'error'"
                variant="flat"
              >
                {{ flow.status }}
              </v-chip>
            </td>
            <td class="text-end text-on-surface-variant md3-body-small">
              {{ flow.last_activity ? new Date(flow.last_activity).toLocaleTimeString() : '—' }}
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card>

    <!-- Flow detail dialog -->
    <v-dialog :model-value="!!selectedFlow" max-width="500" @update:model-value="selectedFlow = null">
      <v-card v-if="selectedFlow">
        <v-card-item class="px-6 pt-6 pb-2">
          <v-card-title class="md3-headline-small pa-0">
            Route Details
          </v-card-title>
        </v-card-item>
        <v-card-text class="px-6">
          <div class="d-flex flex-column ga-2">
            <div>
              <strong class="text-on-surface-variant md3-label-small d-block">SESSION ID</strong>
              <code>{{ selectedFlow.session_id }}</code>
            </div>
            <div>
              <strong class="text-on-surface-variant md3-label-small d-block">SOURCE</strong>
              <span>{{ selectedFlow.source.ip }}:{{ selectedFlow.source.port }}</span>
            </div>
            <div>
              <strong class="text-on-surface-variant md3-label-small d-block">PROTOCOL</strong>
              <span>{{ (selectedFlow.protocol || 'tcp').toUpperCase() }}</span>
            </div>
            <div>
              <strong class="text-on-surface-variant md3-label-small d-block">DESTINATION</strong>
              <span>{{ selectedFlow.destination.domain || selectedFlow.destination.ip }}:{{ selectedFlow.destination.port }}</span>
            </div>
            <div>
              <strong class="text-on-surface-variant md3-label-small d-block">PROXY CHAIN</strong>
              <span v-if="selectedFlow.proxy_chain?.length">
                {{ selectedFlow.proxy_chain.map(p => `${p.name} (${p.type})`).join(' → ') }}
              </span>
              <span v-else>Direct</span>
            </div>
            <div>
              <strong class="text-on-surface-variant md3-label-small d-block">STATUS</strong>
              <v-chip
                size="small"
                :color="selectedFlow.status === 'active' ? 'success' : 'warning'"
                variant="flat"
              >
                {{ selectedFlow.status }}
              </v-chip>
            </div>
          </div>
        </v-card-text>
        <v-card-actions class="px-6 pb-4">
          <v-spacer />
          <v-btn variant="text" @click="selectedFlow = null">Close</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<style scoped>
.live-flow-view {
  padding-bottom: 96px;
}
</style>

