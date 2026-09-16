import type { TFunction } from 'i18next';
import { backendMessageTemplates } from './backend-message-templates';
import { getManualGearLabel, getManualLevelLabel } from './manualGearPresets';

// Canonical state contains backend text or translation instructions, never a rendered string.
export interface DisplayMessage {
  raw?: string;
  key?: string;
  params?: Record<string, unknown>;
  kind?: 'core' | 'bridge' | 'hotkey';
}
export type DisplayMessageInput = DisplayMessage | string;

const escapeRegex = (text: string) => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
const messageMatchers = Object.entries(backendMessageTemplates).map(([key, template]) => {
  const params: string[] = [];
  let source = '';
  let offset = 0;
  for (const match of template.matchAll(/\{\{(\w+)\}\}/g)) {
    source += escapeRegex(template.slice(offset, match.index)) + '([\\s\\S]+?)';
    params.push(match[1]);
    offset = match.index! + match[0].length;
  }
  return { key, params, pattern: new RegExp('^' + source + escapeRegex(template.slice(offset)) + '$') };
});

const CORE_PREFIX = /^(?:核心服务不可用|Core service is unavailable|Core サービスを利用できません)(?=$|[：:。\s])/;
const CORE_REMEDIATION = /。请检查 [^\r\n]+ 是否仍在安装目录中，或是否被安全软件隔离。$/;
const PAWN_IO_HINT = '请在 THRM 内点击“重装 PawnIO”（应用已自带安装包），完成后重启 THRM；若仍不可用，请关闭可能独占硬件传感器的其它监控软件后重试';

export function formatCoreError(raw: string, t: TFunction): string {
  const prefix = raw.trim().match(CORE_PREFIX);
  const detail = prefix
    ? raw.trim().slice(prefix[0].length).replace(CORE_REMEDIATION, '').replace(/^[：:]\s*/, '')
    : raw;
  return detail.trim()
    ? t('store.coreService.unavailableWithDetail', { detail: formatBackendMessage(detail, t) })
    : t('store.coreService.unavailable');
}

// Match complete known messages. Unknown text (including multiline diagnostics) is retained.
export function formatBackendMessage(raw: string, t: TFunction, depth = 0): string {
  if (!raw || depth > 8) return raw;
  const text = raw.trim();
  if (CORE_PREFIX.test(text)) return formatCoreError(text, t);
  if (text === '连接失败') return t('store.errors.connectDevice');
  const unknown = text.match(/^未知(模式)?\((0x[\da-f]+)\)$/i);
  if (unknown) return t(unknown[1] ? 'displayErrors.unknownMode' : 'displayErrors.unknownValue', { code: unknown[2] });

  if (text.endsWith('；' + PAWN_IO_HINT)) {
    const diagnostic = text.slice(0, -('；' + PAWN_IO_HINT).length);
    const cpu = diagnostic.match(/^未读取到 CPU 温度(?:（硬件监控信息: ([\s\S]*)）)?$/);
    const temperatures = diagnostic.match(/^未读取到有效的 CPU\/GPU 温度(?:；硬件监控信息: ([\s\S]*))?$/);
    const fixed = diagnostic === '桥接程序未返回有效温度（CPU/GPU 均为 0）' || diagnostic === 'CPU/GPU 温度读取失败';
    if (cpu || temperatures || fixed) {
      const detail = cpu?.[1] ?? temperatures?.[1];
      return [
        t(cpu ? 'displayErrors.cpuNoTemperature' : 'displayErrors.bridgeNoTemperatures'),
        ...(detail ? [t('displayErrors.hardwareDetail', { detail: formatBackendMessage(detail, t, depth + 1) })] : []),
        t('displayErrors.pawnIoHint'),
      ].join('\n');
    }
  }

  for (const { key, params, pattern } of messageMatchers) {
    const match = text.match(pattern);
    if (!match) continue;
    const values = Object.fromEntries(params.map((name, index) => [name,
      name === 'detail' ? formatBackendMessage(match[index + 1], t, depth + 1) : match[index + 1],
    ]));
    return t('displayErrors.' + key, values);
  }
  const output = text.match(/^([\s\S]+)；输出: ([\s\S]*)$/);
  if (output) {
    return t('displayErrors.installerOutput', {
      detail: formatBackendMessage(output[1], t, depth + 1), output: output[2],
    });
  }
  return raw;
}

// Only the reserved built-in identity is recognizable. Never translate a name by text alone.
export function getProfileDisplayName(profile: { id: string; name: string } | null | undefined, t: TFunction): string {
  if (!profile) return '';
  return profile.id === 'default' && profile.name === '默认' ? t('displayErrors.defaultProfile') : profile.name;
}

export function formatDisplayMessage(message: DisplayMessageInput | null | undefined, t: TFunction): string {
  if (message == null) return '';
  if (typeof message === 'string') return formatBackendMessage(message, t);
  const raw = message.raw ?? '';
  if (message.kind === 'core') return formatCoreError(raw, t);
  if (message.kind === 'bridge' && !raw) return t('store.bridgeWarning.default');
  if (!message.key) return formatBackendMessage(raw, t);

  const params = { ...message.params };
  // These fields contain backend diagnostics, unlike user names, paths and device identifiers.
  for (const name of ['error', 'warning', 'detail', 'message']) {
    if (typeof params[name] === 'string') params[name] = formatBackendMessage(params[name], t);
  }
  if (message.kind === 'hotkey') {
    if (typeof params.gear === 'string') params.gear = getManualGearLabel(params.gear, t);
    if (typeof params.level === 'string') params.level = getManualLevelLabel(params.level, t);
    if (typeof params.profile === 'string' && typeof params.profileId === 'string') {
      params.profile = getProfileDisplayName({ id: params.profileId, name: params.profile }, t);
    }
  }
  return t(message.key, { ...params, defaultValue: formatBackendMessage(raw, t) || message.key });
}

// Diagnostic payloads remain intact. Only named error fields in the displayed copy are localized.
export function formatDebugInfo(info: unknown, t: TFunction): string {
  return JSON.stringify(info, (key, value) =>
    ['error', 'warning', 'lastError', 'bridgeWarning', 'uninstallWarning'].includes(key) && typeof value === 'string'
      ? formatBackendMessage(value, t) : value, 2);
}
