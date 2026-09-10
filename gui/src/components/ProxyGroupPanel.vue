<template>
  <div class="proxy-group-panel">
    <div class="table-wrapper">
      <table class="table is-fullwidth proxy-group-table">
        <caption class="proxy-group-table__caption">
          {{
            $t("proxyGroup.activeGroupLabel")
          }}
        </caption>
        <thead>
          <tr>
            <th class="proxy-group-radio-col" scope="col">
              {{ $t("proxyGroup.active") }}
            </th>
            <th scope="col">{{ $t("proxyGroup.group") }}</th>
            <th class="proxy-group-members-col" scope="col">
              {{ $t("proxyGroup.members") }}
            </th>
            <th class="proxy-group-status-col" scope="col">
              {{ $t("proxyGroup.status") }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="group in groups"
            :key="group"
            :class="{ 'is-active-group': group === activeGroup }"
          >
            <td class="proxy-group-radio-col">
              <label class="proxy-group-radio-label">
                <input
                  v-model="selectedGroup"
                  type="radio"
                  name="active-proxy-group"
                  :value="group"
                  :disabled="saving"
                  :aria-label="$t('proxyGroup.selectActiveGroup', { group })"
                  @change="selectGroup(group)"
                />
              </label>
            </td>
            <th scope="row" class="proxy-group-name-col">
              {{ group.toUpperCase() }}
            </th>
            <td class="proxy-group-members-col">
              {{ memberCount(group) }}
            </td>
            <td class="proxy-group-status-col">
              <b-tag v-if="group === activeGroup" type="is-primary" rounded>
                {{ $t("proxyGroup.active") }}
              </b-tag>
              <span v-else class="has-text-grey">{{
                $t("proxyGroup.inactive")
              }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <p
      v-if="saving"
      class="help is-info proxy-group-panel__saving"
      aria-live="polite"
    >
      {{ $t("proxyGroup.activeGroupSaving") }}
    </p>
  </div>
</template>

<script>
import { handleResponse } from "@/assets/js/utils";

export default {
  name: "ProxyGroupPanel",
  props: {
    groups: {
      type: Array,
      required: true,
    },
    activeGroup: {
      type: String,
      required: true,
    },
    memberCounts: {
      type: Object,
      default: () => ({}),
    },
  },
  data() {
    return {
      selectedGroup: this.activeGroup,
      saving: false,
    };
  },
  watch: {
    activeGroup(value) {
      this.selectedGroup = value;
    },
    groups(groups) {
      if (!groups.includes(this.selectedGroup)) {
        this.selectedGroup = this.activeGroup;
      }
    },
  },
  methods: {
    memberCount(group) {
      return this.memberCounts[group] || 0;
    },
    selectGroup(group) {
      if (this.saving || group === this.activeGroup) {
        return;
      }
      this.saving = true;
      this.$axios({
        url: apiRoot + "/outbound/active",
        method: "put",
        data: { activeOutboundName: group },
      })
        .then((res) => {
          handleResponse(
            res,
            this,
            () => {
              this.$emit("selected", res.data.data.activeOutboundName);
              this.$buefy.toast.open({
                message: this.$t("proxyGroup.activeGroupUpdated"),
                type: "is-primary",
                position: "is-top",
                queue: false,
              });
            },
            () => {
              this.selectedGroup = this.activeGroup;
            }
          );
        })
        .catch(() => {
          this.selectedGroup = this.activeGroup;
        })
        .finally(() => {
          this.saving = false;
        });
    },
  },
};
</script>

<style scoped>
.proxy-group-panel .table-wrapper {
  overflow-x: auto;
  border: 1px solid #edf0f4;
  border-radius: 8px;
}

.proxy-group-table {
  margin-bottom: 0;
  background: transparent;
}

.proxy-group-table__caption {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.proxy-group-table th,
.proxy-group-table td {
  vertical-align: middle;
}

.proxy-group-radio-col {
  width: 4rem;
  text-align: center;
}

.proxy-group-radio-label {
  display: inline-flex;
  padding: 0.35rem;
  cursor: pointer;
}

.proxy-group-radio-label input {
  cursor: inherit;
}

.proxy-group-name-col {
  font-weight: 600;
}

.proxy-group-members-col,
.proxy-group-status-col {
  width: 8rem;
}

.proxy-group-table tr.is-active-group > th,
.proxy-group-table tr.is-active-group > td {
  background: rgba(80, 109, 164, 0.08);
}

.proxy-group-panel__saving {
  margin: 0.75rem 0 0;
}

@media screen and (max-width: 480px) {
  .proxy-group-members-col,
  .proxy-group-status-col {
    width: auto;
  }
}
</style>
