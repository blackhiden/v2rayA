<script setup lang="ts">
// Bulk remove nodes by latency threshold or errors from a proxy group.
// Shows preview and prevents emptying the group entirely.
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { mdiAlertCircle, mdiAlert, mdiCheckCircle } from "@mdi/js";
import { bulkRemoveByLatency, getOutbounds, getTouch } from "@/api";
import type { Touch, TouchSubscription } from "@/api/types";
import { errorText } from "@/api/errors";
import { useNotify } from "@/composables";
import { useAppStore } from "@/stores/app";
import type { Row } from "@/views/nodes/model";

defineOptions({ name: "BulkRemoveByLatencyDialog" });

interface Props {
  source?: string;
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

const connectedInGroup = computed(() => {
  const existingInGroup = new Set(
    store.connectedServer
      .filter((w) => (w.outbound || "proxy") === selectedOutbound.value)
      .map((w) => `${w._type}-${w.id}-${w.sub || 0}`),
  );
  return getAllServers().filter((s) =>
    existingInGroup.has(`${s._type}-${s.id}-${s.sub || 0}`),
  );
});

const serversToRemove = computed(() => {
  return connectedInGroup.value.filter((s) => {
    if (!s.pingLatency) return true; // No latency = error/timeout
    if (s.pingLatency === "TIMEOUT") return true;
    if (!s.pingLatency.endsWith("ms")) return true;
    const latency = parseInt(s.pingLatency, 10);
    return !isNaN(latency) && latency >= maxLatency.value;
  });
});

const wouldEmptyGroup = computed(() => {
  return (
    connectedInGroup.value.length > 0 &&
    serversToRemove.value.length >= connectedInGroup.value.length
  );
});

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
  if (!serversToRemove.value.length || wouldEmptyGroup.value) return;
  saving.value = true;
  try {
    const touches = serversToRemove.value.map((s) => ({
      id: s.id,
      _type: s._type,
      sub: s._type === "subscriptionServer" ? (s.sub ?? 0) : 0,
    }));

    await bulkRemoveByLatency({
      outbound: selectedOutbound.value,
      maxLatencyMs: maxLatency.value,
      serversToCheck: touches,
    });

    notify.success(
      t("bulkRemoveByLatency.success", {
        count: serversToRemove.value.length,
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
        {{ t("bulkRemoveByLatency.title") }}
      </v-card-title>
    </v-card-item>
    <v-card-text class="px-6">
      <v-alert type="info" variant="tonal" density="compact" class="mb-4">
        {{ t("bulkRemoveByLatency.messages[0]") }}
      </v-alert>

      <v-text-field
        v-model.number="maxLatency"
        :label="t('bulkRemoveByLatency.maxLatency')"
        type="number"
        :min="1"
        :max="99999"
        suffix="ms"
        class="mb-2"
      />

      <v-select
        v-model="selectedOutbound"
        :label="t('bulkRemoveByLatency.targetGroup')"
        :items="
          outbounds.map((o) => ({ value: o, title: o.toUpperCase() }))
        "
        class="mb-2"
      />

      <v-alert
        v-if="serversToRemove.length > 0"
        type="error"
        variant="tonal"
        density="compact"
        :icon="mdiAlert"
      >
        {{
          t("bulkRemoveByLatency.preview", {
            count: serversToRemove.length,
            group: selectedOutbound.toUpperCase(),
          })
        }}
        <v-alert
          v-if="wouldEmptyGroup"
          type="warning"
          variant="tonal"
          density="compact"
          :icon="mdiAlertCircle"
          class="mt-2"
        >
          {{ t("bulkRemoveByLatency.wouldEmptyGroup") }}
        </v-alert>
        <div class="mt-2" style="max-height: 160px; overflow-y: auto;">
          <v-list density="compact" bg-color="transparent">
            <v-list-item
              v-for="s in serversToRemove"
              :key="`${s._type}-${s.id}-${s.sub || 0}`"
              :title="s.name || s.address"
              density="compact"
              class="px-1"
            >
              <template #append>
                <span
                  :class="s.pingLatency === 'TIMEOUT' || !s.pingLatency ? 'text-warning' : 'text-error'"
                  class="md3-label-medium"
                >
                  {{ s.pingLatency || 'Timeout/Error' }}
                </span>
              </template>
            </v-list-item>
          </v-list>
        </div>
      </v-alert>

      <v-alert
        v-else-if="hasLatencyData"
        type="success"
        variant="tonal"
        density="compact"
        :icon="mdiCheckCircle"
      >
        {{
          t("bulkRemoveByLatency.previewNoMatch", { threshold: maxLatency })
        }}
      </v-alert>

      <v-alert
        v-else
        type="warning"
        variant="tonal"
        density="compact"
        :icon="mdiAlertCircle"
      >
        {{ t("bulkRemoveByLatency.noLatencyData") }}
      </v-alert>
    </v-card-text>
    <v-card-actions class="px-6 pb-4">
      <v-spacer />
      <v-btn variant="text" @click="emit('close')">
        {{ t("operations.cancel") }}
      </v-btn>
      <v-btn
        variant="flat"
        color="error"
        :disabled="serversToRemove.length === 0 || wouldEmptyGroup"
        :loading="saving"
        @click="confirm"
      >
        {{ t("bulkRemoveByLatency.confirm") }} ({{ serversToRemove.length }})
      </v-btn>
    </v-card-actions>
  </v-card>
</template>

