<template>
  <el-dialog
    custom-class="select-model-dialog"
    :visible.sync="dialogVisible"
    width="600px"
    append-to-body
    :before-close="handleClose"
  >
    <template #title>
      <p class="dialog-title">
        {{ $t('modelExprience.selectModel') }}
        <span class="tip" v-show="!['replace', 'create'].includes(mode)">
          {{
            $t('modelExprience.tip.maxSelectModel').replace(
              '@',
              MAX_SELECT_MODEL,
            )
          }}
        </span>
      </p>
    </template>
    <div class="dialog-body">
      <div class="tool">
        <el-select
          v-model="provider"
          :placeholder="$t('modelExprience.tip.modelSupplier')"
        >
          <el-option
            v-for="item in modelSupplyOptions"
            :key="item.key"
            :label="item.name"
            :value="item.key"
          ></el-option>
        </el-select>
        <el-input
          clearable
          :placeholder="$t('modelExprience.tip.selectModelPlaceholder')"
          suffix-icon="el-icon-search"
          v-model="keyword"
        ></el-input>
      </div>
      <el-empty
        class="noData"
        v-if="!(modelList && modelList.length)"
        :description="$t('common.noData')"
      ></el-empty>
      <div class="model-list" v-else>
        <div
          class="model-row"
          :class="[
            checkIsSelected(item.modelId) && 'active',
            checkIsDisabled(item.modelId) && 'disabled',
          ]"
          v-for="(item, index) in modelList"
          :key="index"
          @click="handleModelSelect(item.modelId)"
        >
          <img class="avatar" :src="getModelAvatar(item.avatar)" />
          <div class="info">
            <p class="title">{{ item.displayName }}</p>
            <p>{{ item.modelDesc || '-' }}</p>
          </div>
          <el-checkbox
            @click.stop
            @change="handleModelSelect(item.modelId)"
            :disabled="checkIsDisabled(item.modelId)"
            :value="checkIsSelected(item.modelId)"
            class="btn"
          ></el-checkbox>
        </div>
      </div>
    </div>
    <div class="dialog-footer">
      <el-button @click="handleClose">
        {{ $t('common.confirm.cancel') }}
      </el-button>
      <el-button type="primary" @click="handleSubmit">
        {{ $t('common.confirm.confirm') }}
      </el-button>
    </div>
  </el-dialog>
</template>
<script>
import { PROVIDER_TYPE } from '@/views/modelAccess/constants';
import { avatarSrc, getModelDefaultIcon } from '@/utils/util';

export default {
  name: 'SelectModelDialog',
  props: {
    modelOptions: {
      // 可选模型列表
      type: Array,
      default: () => [],
    },
  },
  data() {
    return {
      MAX_SELECT_MODEL: 4,
      mode: 'add', // replace: 替换模型； create：创建模型体验； add：添加模型
      dialogVisible: false,
      loading: false,
      keyword: '',
      provider: 'all',
      modelIds: [],
      disabledModelIds: [],
      modelSupplyOptions: [
        {
          key: 'all',
          name: this.$t('modelExprience.all'),
        },
        ...PROVIDER_TYPE,
      ],
    };
  },
  computed: {
    modelList() {
      const reg = new RegExp(this.keyword, 'i');
      const providerReg = new RegExp(
        this.provider === 'all' ? '' : this.provider,
        'i',
      );
      return this.modelOptions.filter(
        item => reg.test(item.displayName) && providerReg.test(item.provider),
      );
    },
    checkIsDisabled() {
      const disabledModelSet = new Set(this.disabledModelIds);
      return modelId => {
        if (this.mode !== 'replace') {
          if (disabledModelSet.has(modelId)) {
            return true;
          }
          if (this.modelIds.includes(modelId)) {
            return false;
          }
          return (
            this.modelIds.length + this.disabledModelIds.length >=
            this.MAX_SELECT_MODEL
          );
        }
        return disabledModelSet.has(modelId);
      };
    },
    checkIsSelected() {
      const modelSet = new Set([...this.modelIds, ...this.disabledModelIds]);
      return modelId => modelSet.has(modelId);
    },
  },
  methods: {
    handleClose() {
      this.dialogVisible = false;
      this.keyword = '';
      this.provider = 'all';
      this.disabledModelIds = [];
      this.modelIds = [];
    },
    openDialog(params) {
      const { mode = 'add', current, disabledSelected } = params || {};
      this.dialogVisible = true;
      this.mode = mode;
      this.modelIds = current || [];
      this.disabledModelIds = disabledSelected || [];
      this.originalModelIds = [...this.modelIds];
    },
    handleSubmit() {
      if (!this.modelIds.length) {
        this.$message.error(this.$t('modelExprience.warning.minSelectModel'));
        return;
      }
      const modelInfoList = [];
      const modelIdMap = new Map();
      this.modelList.forEach(item => {
        modelIdMap.set(item.modelId, item);
      });
      this.modelIds.forEach(id => {
        modelInfoList.push(modelIdMap.get(id));
      });
      this.$emit('submit', this.modelIds, {
        mode: this.mode,
        modelList: modelInfoList,
        originalModelIds: this.originalModelIds,
      }); // this.originalModelIds用于替换模型时，记录原始选择的模型ID
      this.handleClose();
    },
    getModelAvatar(avatar) {
      if (!avatar || !avatar.path) {
        return getModelDefaultIcon();
      }
      return avatarSrc(avatar.path);
    },
    handleModelSelect(modelId) {
      if (this.checkIsDisabled(modelId)) {
        return;
      }
      if (['create', 'replace'].includes(this.mode)) {
        if (this.modelIds.includes(modelId)) {
          return;
        } // 替换模型、创建模型体验时，只能选择一个
        this.modelIds = [modelId];
      } else {
        this.modelIds = this.checkIsSelected(modelId)
          ? this.modelIds.filter(item => item !== modelId)
          : [...this.modelIds, modelId];
      }
    },
  },
};
</script>
<style lang="scss" scoped>
.select-model-dialog {
  .dialog-title {
    font-size: 18px;
    color: #434c6c;
    font-weight: bold;
    .tip {
      font-size: 13px;
      font-weight: normal;
    }
  }
  .dialog-body {
    max-height: 600px;
    min-height: 400px;
    padding: 0 20px;
    display: flex;
    flex-direction: column;
    .tool {
      display: flex;
      gap: 12px;
      margin-bottom: 16px;
      .el-select {
        width: 250px;
      }
    }
    .model-list {
      overflow: auto;
      flex: 1;
      .model-row {
        display: flex;
        align-items: center;
        gap: 12px;
        border: 1px solid #d3d7dd;
        border-radius: 8px;
        padding: 8px 12px;
        margin-bottom: 12px;
        cursor: pointer;
        &.active {
          border-color: $border_color;
          background-color: #f2f7ff;
        }
        &.disabled {
          cursor: not-allowed;
        }
        .avatar {
          flex-shrink: 0;
          width: 40px;
          height: 40px;
          border-radius: 4px;
        }
        .btn {
          flex-shrink: 0;
        }
        .info {
          flex: 1;
          overflow: hidden;

          .title {
            font-weight: 500;
          }
          p {
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
        }
      }
    }
  }
  .dialog-footer {
    text-align: right;
    margin-top: 30px;
  }
}
</style>
