<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  mdiCheck,
  mdiCogOutline,
  mdiDeleteOutline,
  mdiPlus,
  mdiRadioboxBlank,
  mdiRadioboxMarked,
} from "@mdi/js";
import { putActiveOutbound } from "@/api";
import { errorText } from "@/api/errors";
import { useDialog, useNotify, useOutboundGroups } from "@/composables";
import OutboundGroupDialog from "@/dialogs/OutboundGroup.vue";
import { useAppStore } from "@/stores/app";

defineOptions({ name: "ProxyGroupPanel" });

const emit = defineEmits<{ close: [] }>();

const { t } = useI18n();
const notify = useNotify();
const store = useAppStore();
const { open: openDialog } = useDialog();
const groups = useOutboundGroups();

const saving = ref(false);

const memberCounts = computed(() => {
  const counts: Record<string, number> = {};
  for (const group of store.outbounds) {
    counts[group] = store.connectedServer.filter(
      (w) => (w.outbound || "proxy") === group,
    ).length;
  }
  return counts;
});

async function selectActiveGroup(group: string) {
  if (saving.value || group === store.activeOutboundName) return;
  saving.value = true;
  const previous = store.activeOutboundName;
  store.activeOutboundName = group;
  try {
    const res = await putActiveOutbound({ activeOutboundName: group });
    store.activeOutboundName = res.activeOutboundName;
    notify.success(t("proxyGroup.activeGroupUpdated"));
  } catch (err) {
    store.activeOutboundName = previous;
    notify.warning(errorText(err));
  } finally {
    saving.value = false;
  }
}

function openSettings(outbound: string) {
  openDialog(OutboundGroupDialog, { outbound }, { width: 440 });
}

async function removeGroup(outbound: string) {
  await groups.remove(outbound);
}

const add = () => groups.add();
</script>

<template>
  <v-card>
    <v-card-item class="px-6 pt-6 pb-2">
      <div class="d-flex align-center">
        <v-card-title class="md3-headline-small pa-0">
          {{ t("common.proxyGroups") }}
        </v-card-title>
        <v-spacer />
        <v-btn
          variant="tonal"
          color="primary"
          size="small"
          :prepend-icon="mdiPlus"
          @click="add"
        >
          {{ t("operations.create") }}
        </v-btn>
      </div>
      <p class="md3-body-small text-on-surface-variant mt-1 mb-0">
        {{ t("proxyGroup.activeGroupLabel") }}
      </p>
    </v-card-item>

    <v-card-text class="px-6">
      <v-table hover density="comfortable">
        <thead>
          <tr>
            <th class="text-center" style="width: 56px">
              {{ t("proxyGroup.active") }}
            </th>
            <th>{{ t("proxyGroup.group") }}</th>
            <th class="text-center" style="width: 100px">
              {{ t("proxyGroup.members") }}
            </th>
            <th class="text-center" style="width: 100px">
              {{ t("proxyGroup.status") }}
            </th>
            <th class="text-end" style="width: 96px">
              {{ t("common.setting") }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="group in store.outbounds"
            :key="group"
            :class="{
              'bg-primary-container-low': group === store.activeOutboundName,
            }"
          >
            <td class="text-center">
              <v-btn
                variant="text"
                icon
                size="small"
                :disabled="saving"
                :color="group === store.activeOutboundName ? 'primary' : undefined"
                :aria-label="t('proxyGroup.selectActiveGroup', { group })"
                @click="selectActiveGroup(group)"
              >
                <v-icon
                  :icon="
                    group === store.activeOutboundName
                      ? mdiRadioboxMarked
                      : mdiRadioboxBlank
                  "
                />
              </v-btn>
            </td>
            <td class="font-weight-medium">
              {{ group.toUpperCase() }}
            </td>
            <td class="text-center">
              {{ memberCounts[group] || 0 }}
            </td>
            <td class="text-center">
              <v-chip
                v-if="group === store.activeOutboundName"
                size="small"
                color="primary"
                variant="flat"
              >
                {{ t("proxyGroup.active") }}
              </v-chip>
              <span v-else class="text-on-surface-variant md3-label-small">
                {{ t("proxyGroup.inactive") }}
              </span>
            </td>
            <td class="text-end">
              <v-btn
                :icon="mdiCogOutline"
                variant="text"
                size="small"
                :aria-label="t('common.setting')"
                @click="openSettings(group)"
              />
              <v-btn
                v-if="group !== 'proxy'"
                :icon="mdiDeleteOutline"
                variant="text"
                size="small"
                color="error"
                :aria-label="t('operations.delete')"
                @click="removeGroup(group)"
              />
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card-text>

    <v-card-actions class="px-6 pb-4">
      <v-spacer />
      <v-btn variant="text" @click="emit('close')">
        {{ t("operations.close") }}
      </v-btn>
    </v-card-actions>
  </v-card>
</template>

