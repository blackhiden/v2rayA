<script setup lang="ts">
// Bulk add nodes by latency threshold to a proxy group. Shows a preview
// of matching nodes and adds them while preserving existing connections
// from all subscriptions.
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { mdiAlertCircle, mdiCheckCircle } from "@mdi/js";
import { getOutbounds, putOutboundConnections } from "@/api";
import type { Which } from "@/api/types";
import { errorText } from "@/api/errors";
import { useNotify } from "@/composables";
import { useAppStore } from "@/stores/app";
import { useNodes, type Row } from "@/views/nodes/model";

defineOptions({ name: "BulkAddByLatencyDialog" });

interface Props {
  source?: string; // "local" or subscription address
}

const props = withDefaults(defineProps<Props>(), {
  source: "all",
});

const emit = defineEmits<{ close: [success?: boolean] }>();

const { t } = useI18n();
const notify = useNotify();
const store = useAppStore();
const nodes = useNodes();

const maxLatency = ref(500);
const selectedOutbound = ref("proxy");
const outbounds = ref<string[]>(["proxy"]);
const saving = ref(false);

const hasLatencyData = computed(() => {
  return getAllServers().some(
    (s) => s.pingLatency && s.pingLatency.endsWith("ms"),
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

function getAllServers(): Row[] {
  const touch = nodes.touch.value;
  if (props.source === "local") {
    return touch.servers;
  } else if (props.source === "all") {
    return [
      ...touch.servers,
      ...touch.subscriptions.flatMap((s) => s.servers),
    ];
  } else {
    const sub = touch.subscriptions.find(
      (s) => s.address === props.source,
    );
    return sub ? sub.servers : [];
  }
}

onMounted(async () => {
  try {
    const res = await getOutbounds();
    outbounds.value = res.outbounds;
    if (!outbounds.value.includes(selectedOutbound.value)) {
      selectedOutbound.value = outbounds.value[0] || "proxy";
    }
  } catch (err) {
    notify.warning(errorText(err));
  }
});

async function confirm() {
  if (!serversToAdd.value.length) return;
  saving.value = true;
  try {
    const existingTouches = store.connectedServer
      .filter((w) => (w.outbound || "proxy") === selectedOutbound.value)
      .map((w) => ({
        id: w.id,
        _type: w._type,
        sub: w._type === "subscriptionServer" ? w.sub ?? 0 : 0,
        outbound: selectedOutbound.value,
      }));

    const newTouches = serversToAdd.value.map((s) => ({
      id: s.id,
      _type: s._type,
      sub: s._type === "subscriptionServer" ? s.sub ?? 0 : 0,
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
