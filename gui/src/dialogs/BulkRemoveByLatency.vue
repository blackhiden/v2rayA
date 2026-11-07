<script setup lang="ts">
// Bulk remove nodes by latency threshold or errors from a proxy group.
// Shows preview and prevents emptying the group entirely.
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { mdiAlertCircle, mdiAlert, mdiCheckCircle } from "@mdi/js";
import { bulkRemoveByLatency, getOutbounds } from "@/api";
import type { Which } from "@/api/types";
import { errorText } from "@/api/errors";
import { useNotify } from "@/composables";
import { useAppStore } from "@/stores/app";
import { useNodes, type Row } from "@/views/nodes/model";

defineOptions({ name: "BulkRemoveByLatencyDialog" });

interface Props {
  source?: string;
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

const connectedInGroup = computed(() => {
  return store.connectedServer
    .filter((w) => (w.outbound || "proxy") === selectedOutbound.value)
    .map((w) => {
      const servers = getAllServers();
      return servers.find(
        (s) => s.id === w.id && s._type === w._type && (s.sub ?? 0) === (w.sub ?? 0),
      );
    })
    .filter((s): s is Row => s !== null && s !== undefined);
});

const serversToRemove = computed(() => {
  return connectedInGroup.value.filter((s) => {
    if (!s.pingLatency) return true; // No latency = error/timeout
    if (s.pingLatency === "TIMEOUT") return true;
    if (!s.pingLatency.endsWith("ms")) return true;
    const latency = parseInt(s.pingLatency, 10);
    return !isNaN(latency) && latency > maxLatency.value;
  });
});

const wouldEmptyGroup = computed(() => {
  return (
    connectedInGroup.value.length > 0 &&
    serversToRemove.value.length >= connectedInGroup.value.length
  );
});

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
  if (!serversToRemove.value.length || wouldEmptyGroup.value) return;
  saving.value = true;
  try {
    const touches = serversToRemove.value.map((s) => ({
      id: s.id,
      _type: s._type,
      sub: s._type === "subscriptionServer" ? s.sub ?? 0 : 0,
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

