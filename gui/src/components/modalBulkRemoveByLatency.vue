<template>
  <div class="modal-card" style="max-width: 600px; margin: auto">
    <header class="modal-card-head">
      <p class="modal-card-title">{{ $t("bulkRemoveByLatency.title") }}</p>
    </header>
    <section class="modal-card-body">
      <b-message type="is-info" size="is-small" :closable="false">
        {{ $t("bulkRemoveByLatency.messages[0]") }}
      </b-message>
      <b-field :label="$t('bulkRemoveByLatency.maxLatency')">
        <b-numberinput v-model="maxLatency" :min="1" :max="99999" :step="50"
          :placeholder="$t('bulkRemoveByLatency.maxLatencyPlaceholder')" controls-position="compact" expanded />
      </b-field>
      <b-field :label="$t('bulkRemoveByLatency.targetGroup')">
        <b-select v-model="selectedOutbound" expanded>
          <option v-for="outbound in outbounds" :key="outbound" :value="outbound">
            {{ outbound.toUpperCase() }}
          </option>
        </b-select>
      </b-field>
      <div class="bulk-remove-preview">
        <div v-if="serversToRemove.length > 0" class="has-text-danger">
          <b-icon icon="alert-triangle" size="is-small" />
          {{ $t("bulkRemoveByLatency.preview", { count: serversToRemove.length, group: selectedOutbound.toUpperCase() }) }}
          <div v-if="wouldEmptyGroup" class="has-text-warning" style="margin-top:4px;font-size:0.85em">
            <b-icon icon="alert-circle" size="is-small" />
            {{ $t("bulkRemoveByLatency.wouldEmptyGroup") }}
          </div>
        </div>
        <div v-else-if="hasLatencyData" class="has-text-success">
          <b-icon icon="check-circle" size="is-small" />
          {{ $t("bulkRemoveByLatency.previewNoMatch", { threshold: maxLatency }) }}
        </div>
        <div v-else class="has-text-warning">
          <b-icon icon="alert" size="is-small" />
          {{ $t("bulkRemoveByLatency.noLatencyData") }}
        </div>
        <div v-if="serversToRemove.length > 0" class="bulk-remove-server-list">
          <div v-for="server in serversToRemove" :key="server.id + server._type + (server.sub || 0)"
            class="bulk-remove-server-item">
            <div class="bulk-remove-server-name">{{ server.name || server.host || server.address }}</div>
            <div class="bulk-remove-server-reason">
              <span v-if="server.reason === 'latency'" class="has-text-danger">
                {{ server.pingLatency || 'N/A' }}
              </span>
              <span v-else class="has-text-warning">
                {{ server.pingLatency || 'Error/Timeout' }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </section>
    <footer class="modal-card-foot" style="justify-content:flex-end">
      <b-button @click="cancel">{{ $t("operations.cancel") }}</b-button>
      <b-button type="is-danger" :disabled="serversToRemove.length===0 || wouldEmptyGroup" :loading="saving" @click="confirm">
        {{ $t("bulkRemoveByLatency.confirm") }} ({{ serversToRemove.length }})
      </b-button>
    </footer>
  </div>
</template>

<script>
import axios from "../plugins/axios";
const apiRoot = process.env.VUE_APP_API_ROOT || "/api";

export default {
  name: "ModalBulkRemoveByLatency",
  props: {
    tabType: { type: String, default: "all" },
    tabIndex: { type: Number, default: -1 },
  },
  data() {
    return {
      maxLatency: 500,
      selectedOutbound: "proxy",
      outbounds: ["proxy"],
      touchData: null,
      currentConnectedServers: [],
      saving: false,
    };
  },
  computed: {
    hasLatencyData() {
      if (!this.touchData) return false;
      return (this.touchData.servers || []).some((s) => s.pingLatency && s.pingLatency.endsWith("ms")) ||
             (this.touchData.subscriptions || []).some((sub) =>
               (sub.servers || []).some((s) => s.pingLatency && s.pingLatency.endsWith("ms"))
             );
    },
    matchingServers() {
      if (!this.touchData) return [];
      return this.getAllServers().filter((s) => {
        if (s.disableReason) return false;
        if (!s.pingLatency || !s.pingLatency.endsWith("ms")) {
          return true;
        }
        const latency = parseInt(s.pingLatency, 10);
        return !isNaN(latency) && latency >= this.maxLatency;
      });
    },
    serversToRemove() {
      if (!this.matchingServers || !this.matchingServers.length) return [];
      const existingInGroup = (this.currentConnectedServers || []).filter(
        (w) => (w.outbound || "proxy") === this.selectedOutbound
      ).map((w) => `${w._type || w.TYPE}-${w.id || w.ID}-${w.sub || 0}`);

      return this.matchingServers.map((s) => {
        const key = `${s._type || s.TYPE}-${s.id || s.ID}-${s.sub || 0}`;
        const isInGroup = existingInGroup.includes(key);
        const pingLatency = s.pingLatency || "";
        return {
          ...s,
          inGroup: isInGroup,
          reason: !pingLatency || !pingLatency.endsWith("ms") ? "error" : "latency",
        };
      }).filter((s) => s.inGroup);
    },
    wouldEmptyGroup() {
      const groupServers = (this.currentConnectedServers || []).filter(
        (w) => (w.outbound || "proxy") === this.selectedOutbound
      );
      if (!groupServers || groupServers.length === 0) return false;

      const serversToRemoveSet = new Set(
        (this.serversToRemove || []).map((s) => `${s._type || s.TYPE}-${s.id || s.ID}-${s.sub || 0}`)
      );
      const remaining = groupServers.filter((w) => {
        const key = `${w._type || w.TYPE}-${w.id || w.ID}-${w.sub || 0}`;
        return !serversToRemoveSet.has(key);
      });
      return remaining.length === 0;
    },
  },
  methods: {
    getAllServers() {
      if (!this.touchData) return [];
      if (this.tabType === "server") {
        return (this.touchData.servers || []).map((s) => ({ ...s, _type: "server" }));
      }
      if (this.tabType === "subscription" && this.tabIndex >= 0
          && this.tabIndex < (this.touchData.subscriptions || []).length) {
        const sub = this.touchData.subscriptions[this.tabIndex];
        return (sub.servers || []).map((s) => ({ ...s, _type: "subscriptionServer", sub: this.tabIndex }));
      }
      const servers = (this.touchData.servers || []).map((s) => ({ ...s, _type: "server" }));
      const subServers = [];
      (this.touchData.subscriptions || []).forEach((sub, subIdx) => {
        (sub.servers || []).forEach((s) => subServers.push({ ...s, _type: "subscriptionServer", sub: subIdx }));
      });
      return [...servers, ...subServers];
    },
    async fetchTouchData() {
      try {
        const res = await axios({ url: apiRoot + "/outbounds" });
        if (res.data.code === "SUCCESS" && res.data.data.outbounds) {
          this.outbounds = res.data.data.outbounds;
          if (!this.outbounds.includes(this.selectedOutbound)) {
            this.selectedOutbound = this.outbounds[0] || "proxy";
          }
        }
      } catch (e) { /* ignore */ }
      try {
        const res = await axios({ url: apiRoot + "/touch" });
        if (res.data.code === "SUCCESS") {
          this.touchData = res.data.data.touch;
          this.currentConnectedServers = res.data.data.touch.connectedServer || [];
        }
      } catch (e) { /* ignore */ }
    },
    cancel() {
      this.$emit("close");
    },
    async confirm() {
      if (!this.serversToRemove.length || this.wouldEmptyGroup) return;
      this.saving = true;
      try {
        const touches = this.serversToRemove.map((s) => ({
          id: s.id || s.ID,
          _type: s._type || s.TYPE,
          sub: s._type === "subscriptionServer" ? (s.sub != null ? s.sub : 0) : 0,
        }));
        const res = await axios({
          url: apiRoot + "/bulkRemoveByLatency",
          method: "post",
          data: {
            outbound: this.selectedOutbound,
            maxLatencyMs: this.maxLatency,
            serversToCheck: touches,
          },
        });
        if (res.data.code === "SUCCESS") {
          this.$buefy.toast.open({
            type: "is-success",
            position: "is-top",
            duration: 5000,
            message: this.$t("bulkRemoveByLatency.success", {
              count: this.serversToRemove.length,
              group: this.selectedOutbound.toUpperCase(),
            }),
          });
          this.$emit("close");
          this.$emit("done");
        } else {
          this.$buefy.toast.open({
            type: "is-warning",
            position: "is-top",
            duration: 5000,
            message: res.data.message || this.$t("common.fail"),
          });
        }
      } catch (err) {
        this.$buefy.toast.open({
          type: "is-warning",
          position: "is-top",
          duration: 5000,
          message: err?.response?.data?.message || err?.message || this.$t("common.fail"),
        });
      } finally {
        this.saving = false;
      }
    },
  },
  async mounted() {
    await this.fetchTouchData();
  },
  watch: {
    tabType() { this.fetchTouchData(); },
    tabIndex() { this.fetchTouchData(); },
  },
};
</script>

<style lang="scss" scoped>
.bulk-remove-preview {
  padding: 10px 12px;
  border-radius: 6px;
  background: #f5f5f5;
  margin-top: 8px;
  font-size: 0.95em;
}
.bulk-remove-server-list {
  max-height: 200px;
  overflow-y: auto;
  margin-top: 8px;
  border: 1px solid #eee;
  border-radius: 4px;
}
.bulk-remove-server-item {
  display: flex;
  align-items: center;
  padding: 4px 10px;
  font-size: 0.85em;
  gap: 6px;
  border-bottom: 1px solid #f5f5f5;
  &:last-child { border-bottom: none; }
}
.bulk-remove-server-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bulk-remove-server-reason { flex-shrink: 0; font-size: 0.8em; }
body.theme-dark {
  .bulk-remove-preview { background: #2b2930; }
  .bulk-remove-server-list { border-color: #444; }
  .bulk-remove-server-item { border-bottom-color: #333; }
}
</style>
