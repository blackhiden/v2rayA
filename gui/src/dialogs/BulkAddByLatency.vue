<script setup lang="ts">
// Bulk add nodes by latency threshold to a proxy group. Shows a preview
// of matching nodes and adds them while preserving existing connections
// from all subscriptions.
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { mdiAlertCircle, mdiCheckCircle } from "@mdi/js";
import { getOutbounds, getTouch, putOutboundConnections } from "@/api";
import type { Touch, TouchSubscription } from "@/api/types";
import { errorText } from "@/api/errors";
import { useNotify } from "@/composables";
import { useAppStore } from "@/stores/app";
import type { Row } from "@/views/nodes/model";

defineOptions({ name: "BulkAddByLatencyDialog" });

interface Props {
  source?: string; // "all", "local", or subscription address
  rows?: Row[];
  subscriptions?: TouchSubscription[];
}

const props = withDefaults(defineProps<Props>(), {
  source: "all",
  rows: () => [],
  subscriptions: () => [],
});

const emit = defineEmits<{ close: [success?: boolean] }>();

const { t } = useI18n();
const notify = useNotify();
const store = useAppStore();

const maxLatency = ref(500);
const selectedOutbound = ref(store.outboundName || "proxy");
const outbounds = ref<string[]>(
  store.outbounds.length ? [...store.outbounds] : ["proxy"],
);
const touchFallback = ref<Touch | null>(null);
const saving = ref(false);

function getAllServers(): Row[] {
  let all: Row[];
  if (props.rows && props.rows.length > 0) {
    all = props.rows;
  } else if (touchFallback.value) {
    all = [
      ...touchFallback.value.servers,
      ...touchFallback.value.subscriptions.flatMap((s) => s.servers),
    ];
  } else {
    all = [];
  }

  if (props.source === "local") {
    return all.filter((r) => r._type === "server");
  } else if (props.source === "all") {
    return all;
  } else {
    const subs = props.subscriptions.length
      ? props.subscriptions
      : touchFallback.value?.subscriptions ?? [];
    const subIdx = subs.findIndex((s) => s.address === props.source);
    if (subIdx >= 0) {
      return all.filter(
        (r) => r._type === "subscriptionServer" && r.sub === subIdx,
      );
    }
    return all;
  }
}

const hasLatencyData = computed(() => {
  return getAllServers().some(
    (s) =>
      s.pingLatency &&
      (s.pingLatency.endsWith("ms") || s.pingLatency === "TIMEOUT"),
  );
});

const matchingServers = computed(() => {
  return getAllServers().filter((s) => {
    if (s.disableReason) return false;
    if (!s.pingLatency || !s.pingLatency.endsWith("ms")) return false;
    const latency = parseInt(s.pingLatency, 10);
    return !isNaN(latency) && latency <= maxLatency.value;
  });
});

const serversToAdd = computed(() => {
  if (!matchingServers.value.length) return [];
  const existingKeys = new Set(
    store.connectedServer
      .filter((w) => (w.outbound || "proxy") === selectedOutbound.value)
      .map((w) => `${w._type}-${w.id}-${w.sub || 0}`),
  );
  return matchingServers.value.filter(
    (s) => !existingKeys.has(`${s._type}-${s.id}-${s.sub || 0}`),
  );
});

const alreadyInGroupCount = computed(
  () => matchingServers.value.length - serversToAdd.value.length,
);

onMounted(async () => {
  try {
    const res = await getOutbounds();
    if (res?.outbounds?.length) {
      outbounds.value = res.outbounds;
    }
    if (!outbounds.value.includes(selectedOutbound.value)) {
      selectedOutbound.value = outbounds.value[0] || "proxy";
    }
    if (!props.rows || props.rows.length === 0) {
      const touchRes = await getTouch();
      touchFallback.value = touchRes.touch;
      if (!store.connectedServer.length && touchRes.touch.connectedServer) {
        store.connectedServer = touchRes.touch.connectedServer;
      }
    }
  } catch (err) {
    notify.warning(errorText(err));
  }
});

async function confirm() {
  if (!serversToAdd.value.length) return;
  saving.value = true;
  try {
    // Preserve ALL currently connected servers for this outbound
    const existingTouches = store.connectedServer
      .filter((w) => (w.outbound || "proxy") === selectedOutbound.value)
      .map((w) => ({
        id: w.id,
        _type: w._type,
        sub: w._type === "subscriptionServer" ? (w.sub ?? 0) : 0,
        outbound: selectedOutbound.value,
      }));

    const newTouches = serversToAdd.value.map((s) => ({
      id: s.id,
      _type: s._type,
      sub: s._type === "subscriptionServer" ? (s.sub ?? 0) : 0,
      outbound: selectedOutbound.value,
    }));

    await putOutboundConnections({
      outbound: selectedOutbound.value,
      touches: [...existingTouches, ...newTouches],
    });

    notify.success(
      t("bulkAddByLatency.success", {
        count: serversToAdd.value.length,
        group: selectedOutbound.value.toUpperCase(),
      }),
    );
    emit("close", true);
  } catch (err) {
    notify.warning(errorText(err));
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <v-card>
    <v-card-item class="px-6 pt-6 pb-2">
      <v-card-title class="md3-headline-small pa-0">
        {{ t("bulkAddByLatency.title") }}
      </v-card-title>
    </v-card-item>
    <v-card-text class="px-6">
      <v-alert type="info" variant="tonal" density="compact" class="mb-4">
        {{ t("bulkAddByLatency.messages[0]") }}
      </v-alert>

      <v-text-field
        v-model.number="maxLatency"
        :label="t('bulkAddByLatency.maxLatency')"
        type="number"
        :min="1"
        :max="99999"
        suffix="ms"
        class="mb-2"
      />

      <v-select
        v-model="selectedOutbound"
        :label="t('bulkAddByLatency.targetGroup')"
        :items="
          outbounds.map((o) => ({ value: o, title: o.toUpperCase() }))
        "
        class="mb-2"
      />

      <v-alert
        v-if="serversToAdd.length > 0"
        type="success"
        variant="tonal"
        density="compact"
        :icon="mdiCheckCircle"
      >
        {{
          t("bulkAddByLatency.preview", {
            count: serversToAdd.length,
            group: selectedOutbound.toUpperCase(),
          })
        }}
        <div v-if="alreadyInGroupCount > 0" class="text-warning mt-1">
          {{
            t("bulkAddByLatency.alreadyInGroup", {
              count: alreadyInGroupCount,
            })
          }}
        </div>
        <div class="mt-2" style="max-height: 160px; overflow-y: auto;">
          <v-list density="compact" bg-color="transparent">
            <v-list-item
              v-for="s in serversToAdd"
              :key="`${s._type}-${s.id}-${s.sub || 0}`"
              :title="s.name || s.address"
              density="compact"
              class="px-1"
            >
              <template #append>
                <span class="text-primary md3-label-medium">
                  {{ s.pingLatency }}
                </span>
              </template>
            </v-list-item>
          </v-list>
        </div>
      </v-alert>

      <v-alert
        v-else-if="hasLatencyData"
        type="error"
        variant="tonal"
        density="compact"
        :icon="mdiAlertCircle"
      >
        {{
          t("bulkAddByLatency.previewNoMatch", { threshold: maxLatency })
        }}
      </v-alert>

      <v-alert
        v-else
        type="warning"
        variant="tonal"
        density="compact"
        :icon="mdiAlertCircle"
      >
        {{ t("bulkAddByLatency.noLatencyData") }}
      </v-alert>
    </v-card-text>
    <v-card-actions class="px-6 pb-4">
      <v-spacer />
      <v-btn variant="text" @click="emit('close')">
        {{ t("operations.cancel") }}
      </v-btn>
      <v-btn
        variant="flat"
        color="primary"
        :disabled="serversToAdd.length === 0"
        :loading="saving"
        @click="confirm"
      >
        {{ t("bulkAddByLatency.confirm") }} ({{ serversToAdd.length }})
      </v-btn>
    </v-card-actions>
  </v-card>
</template>
