<template>
  <BaseDialog :show="show" :title="t('keys.useKeyModal.title')" width="wide" @close="emit('close')">
    <UseKeyContent :show="show" :api-key="apiKey" :base-url="baseUrl" :platform="platform" :claude-code-only="claudeCodeOnly" :allow-messages-dispatch="allowMessagesDispatch" />
    <template #footer><button class="btn btn-secondary" @click="emit('close')">{{ t('common.close') }}</button></template>
  </BaseDialog>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import UseKeyContent from './UseKeyContent.vue'
import type { GroupPlatform } from '@/types'
defineProps<{ show: boolean; apiKey: string; baseUrl: string; platform: GroupPlatform | null; claudeCodeOnly?: boolean; allowMessagesDispatch?: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
</script>
