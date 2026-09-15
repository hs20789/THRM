import { create } from 'zustand';
import { types } from '../../../wailsjs/go/models';
import { apiService } from '../services/api';
import { configService } from '../services/config-service';
import { deviceService, type DeviceStatusPayload } from '../services/device-service';
import {
  appendSampledHistoryPoint,
  createLiveHistoryPoint,
  mergeTimelineEvents,
  SESSION_HISTORY_LIMIT,
  SESSION_HISTORY_RETENTION_MS,
} from '../lib/temperature-history';
import type { TemperatureHistoryPoint, TimelineEvent } from '../lib/temperature-history';
import { i18n } from '../lib/i18n';
import { getManualGearLabel, getManualLevelLabel } from '../lib/manualGearPresets';
import { toast } from 'sonner';
import type { DeviceSettings } from '../types/app';

const getBridgeWarningMessage = () => i18n.t('store.bridgeWarning.default');

// 核心服务（Go）生成的成品文案。核心常驻后台、不感知 GUI 当前语言，所以这些整句一旦
// 直接渲染，切到日/韩/英的用户就会看到中文。下面按前缀识别后换成本地化文案。
//
// 这些不是协议值，只是"曾经由某个版本的核心发出过"的提示语。历史上核心用过中/英/日三
// 种文案，都要认得；核心侧文案若改动，这里同步补一条即可，认不出时走带 detail 的兜底。
const CORE_SERVICE_UNAVAILABLE_PREFIXES = [
  '核心服务不可用',
  'Core service is unavailable',
  'Core サービスを利用できません',
];

// 核心的"带原因"形态：核心服务不可用：{detail}。请检查 ...（ipc_client.go
// coreServiceUnavailableMessage）。detail 是唯一有信息量的部分，换文案时必须捞出来带走，
// 否则排障线索就没了。捞不到就按无 detail 的形态处理。
const CORE_SERVICE_DETAIL_PATTERN = /^核心服务不可用[：:]\s*(.+?)。请检查/;

// 桥接（Windows 专用）读温失败时核心塞进 BridgeMsg 的两句固定提示，内容等同
// store.bridgeWarning.default。注意只匹配这两句常量；bridgeTemp.Error 那种由 C# 侧
// 生成的任意错误原文不在此列，要原样透传，否则会吞掉唯一的排障线索。
const BRIDGE_WARNING_PREFIXES = [
  '桥接程序未返回有效温度',
  'CPU/GPU 温度读取失败',
];

// 核心在 system_device.go 里广播的设备错误常量。
const DEVICE_CONNECT_FAILED = '连接失败';

const matchesAnyPrefix = (text: string, prefixes: string[]) =>
  prefixes.some((prefix) => text.startsWith(prefix));

// 把后端成品文案换成本地化文案，同时在控制台留下原文。原文不会进 UI，但排障时
// （尤其是只能拿到前端 DevTools 的场景）还看得见。
const localizeBackendMessage = (raw: string, localized: string) => {
  console.warn('[i18n] backend message replaced:', raw);
  return localized;
};

const getCoreServiceErrorMessage = (detail?: string) => {
  const trimmed = detail?.trim();
  if (!trimmed) {
    return i18n.t('store.coreService.unavailable');
  }
  // 核心生成的整句：丢掉外壳换成本地化文案，但要保住里面的 detail。
  const withDetail = trimmed.match(CORE_SERVICE_DETAIL_PATTERN);
  if (withDetail) {
    return localizeBackendMessage(
      trimmed,
      i18n.t('store.coreService.unavailableWithDetail', { detail: withDetail[1] }),
    );
  }
  if (matchesAnyPrefix(trimmed, CORE_SERVICE_UNAVAILABLE_PREFIXES)) {
    return localizeBackendMessage(trimmed, i18n.t('store.coreService.unavailable'));
  }
  // 已经是当前语言的整句（重复处理同一条错误时会走到）：原样返回，避免嵌套包装。
  if (trimmed.includes(i18n.t('store.coreService.unavailable'))) {
    return trimmed;
  }
  // 其余是带信息量的 detail（核心传上来的具体原因），保留并包进本地化外壳。
  return i18n.t('store.coreService.unavailableWithDetail', { detail: trimmed });
};

const getLocalizedBridgeMessage = (bridgeMessage: string) => {
  if (!bridgeMessage) {
    return getBridgeWarningMessage();
  }
  if (matchesAnyPrefix(bridgeMessage, BRIDGE_WARNING_PREFIXES)) {
    return localizeBackendMessage(bridgeMessage, getBridgeWarningMessage());
  }
  // C# 桥接生成的任意错误原文——无法映射，原样显示好过丢失信息。
  return bridgeMessage;
};

const getLocalizedDeviceError = (errorMsg: string) => {
  if (errorMsg?.trim() === DEVICE_CONNECT_FAILED) {
    return localizeBackendMessage(errorMsg, i18n.t('store.errors.connectDevice'));
  }
  return errorMsg;
};

// 核心在 messageParams 里放的 gear/level 是设备协议原始值（"静音"、"中"），不是文案。
// 这里只在插值前翻成当前语言，值本身不动。
const translateHotkeyParams = (params?: Record<string, unknown>) => {
  if (!params) {
    return undefined;
  }
  const translated: Record<string, unknown> = { ...params };
  if (typeof params.gear === 'string') {
    translated.gear = getManualGearLabel(params.gear);
  }
  if (typeof params.level === 'string') {
    translated.level = getManualLevelLabel(params.level);
  }
  return translated;
};

const isCoreServiceFailureDetail = (detail?: string) => {
  const normalized = detail?.toLowerCase() ?? '';
  return normalized.includes('core') ||
    normalized.includes('核心服务') ||
    normalized.includes('ipc') ||
    normalized.includes('服务器') ||
    normalized.includes('服务');
};

type ActiveTab = 'status' | 'curve' | 'control' | 'about';
export type CurveFocusTarget = 'curve-editor' | 'history-details';
// labelKey 为 i18n 键（在 FanCurve 时间线渲染时翻译）；不再存储已本地化的字面量，保证跟随语言切换。
export type { TimelineEvent } from '../lib/temperature-history';

interface AppStore {
  isConnected: boolean;
  deviceProductId: string | null;
  deviceModel: string | null;
  deviceSettings: DeviceSettings | null;
  config: types.AppConfig | null;
  fanData: types.FanData | null;
  temperature: types.TemperatureData | null;
  legionFnQSupported: boolean;
  bridgeWarning: string | null;
  coreServiceError: string | null;
  // 上面两条是渲染用的本地化文案；下面两条是后端原文，只做保留不进 UI，用于排障。
  bridgeWarningRaw: string | null;
  coreServiceErrorRaw: string | null;
  isLoading: boolean;
  error: string | null;
  activeTab: ActiveTab;
  curveFocusTarget: CurveFocusTarget | null;
  sessionHistoryPoints: TemperatureHistoryPoint[];
  timelineEvents: TimelineEvent[];

  setActiveTab: (tab: ActiveTab) => void;
  openCurveTab: (target: CurveFocusTarget) => void;
  clearCurveFocusTarget: () => void;
  clearBridgeWarning: () => void;
  handleTemperaturePayload: (data: types.TemperatureData | null) => void;
  appendSessionHistoryPoint: (data: types.TemperatureData | null) => void;
  mergeTimelineEvents: (events: TimelineEvent[] | null | undefined) => void;

  initializeApp: () => Promise<void>;
  resyncCore: () => Promise<void>;
  connectDevice: () => Promise<void>;
  disconnectDevice: () => Promise<void>;
  updateConfig: (config: types.AppConfig) => Promise<void>;

  startEventListeners: () => () => void;
}

// resyncCore 的重入哨兵，见该方法注释。
let resyncInFlight = false;

export const useAppStore = create<AppStore>((set, get) => ({
  isConnected: false,
  deviceProductId: null,
  deviceModel: null,
  deviceSettings: null,
  config: null,
  fanData: null,
  temperature: null,
  legionFnQSupported: false,
  bridgeWarning: null,
  coreServiceError: null,
  bridgeWarningRaw: null,
  coreServiceErrorRaw: null,
  isLoading: true,
  error: null,
  activeTab: 'status',
  curveFocusTarget: null,
  sessionHistoryPoints: [],
  timelineEvents: [],

  setActiveTab: (tab) => set({ activeTab: tab, curveFocusTarget: null }),

  openCurveTab: (target) => set({ activeTab: 'curve', curveFocusTarget: target }),

  clearCurveFocusTarget: () => set({ curveFocusTarget: null }),

  clearBridgeWarning: () => set({ bridgeWarning: null }),

  handleTemperaturePayload: (data) => {
    const bridgeMessage = data?.bridgeMessage?.trim() ?? '';
    const bridgeFailed = data?.bridgeOk === false;
    set({
      temperature: data,
      bridgeWarning: bridgeFailed ? getLocalizedBridgeMessage(bridgeMessage) : null,
      bridgeWarningRaw: bridgeFailed ? bridgeMessage || null : null,
    });
  },

  appendSessionHistoryPoint: (data) => {
    if (!data) return;

    const point = createLiveHistoryPoint({
      updateTime: data.updateTime,
      cpuTemp: data.cpuTemp,
      gpuTemp: data.gpuTemp,
      cpuPower: data.cpuPower,
      gpuPower: data.gpuPower,
      cpuFanRpm: (data as { cpuFanRpm?: number }).cpuFanRpm,
      gpuFanRpm: (data as { gpuFanRpm?: number }).gpuFanRpm,
    }, Number(get().fanData?.currentRpm || 0));

    if (!point) return;

    set((state) => ({
      sessionHistoryPoints: appendSampledHistoryPoint(state.sessionHistoryPoints, point, {
        retentionMs: SESSION_HISTORY_RETENTION_MS,
        limit: SESSION_HISTORY_LIMIT,
      }),
    }));
  },

  // 快照与实时推送共用一个入口，由 mergeTimelineEvents 负责去重与排序。
  mergeTimelineEvents: (events) => {
    if (!events || events.length === 0) return;
    set((state) => ({ timelineEvents: mergeTimelineEvents(state.timelineEvents, events) }));
  },

  initializeApp: async () => {
    try {
      set({ isLoading: true });

      const [appConfig, deviceStatus, debugInfo] = await Promise.all([
        configService.getConfig(),
        deviceService.getStatus() as Promise<DeviceStatusPayload>,
        apiService.getDebugInfo().catch(() => null),
      ]);
      const coreServiceError = deviceStatus.error ? getCoreServiceErrorMessage(deviceStatus.error) : null;

      set({
        config: appConfig,
        isConnected: deviceStatus.connected || false,
        deviceProductId: deviceStatus.productId || null,
        deviceModel: deviceStatus.model || null,
        deviceSettings: deviceStatus.deviceSettings || null,
        fanData: deviceStatus.currentData || null,
        legionFnQSupported: debugInfo?.legionFnQSupported === true,
        coreServiceError,
        coreServiceErrorRaw: deviceStatus.error || null,
        error: coreServiceError,
      });

      get().handleTemperaturePayload(deviceStatus.temperature || null);
    } catch (error) {
      console.error('初始化失败:', error);
      const detail = error instanceof Error ? error.message : undefined;
      const coreServiceError = isCoreServiceFailureDetail(detail) ? getCoreServiceErrorMessage(detail) : null;
      set({
        error: coreServiceError || i18n.t('store.errors.initializeApp'),
        coreServiceError,
        coreServiceErrorRaw: coreServiceError ? detail || null : null,
      });
    } finally {
      set({ isLoading: false });
    }
  },

  // resyncCore 在 IPC 连接恢复后重新拉取核心状态。
  //
  // 必须有这一步：core-service-error 会把 isConnected/deviceModel/deviceSettings 清空，
  // 而这些字段只能由 device-connected 事件恢复——设备其实从未断开，该事件不会再来。
  // 缺少主动重取时，一次 IPC 抖动就会让界面永久停在"设备未连接"，只能重启 GUI。
  // 与 initializeApp 的区别是不触碰 isLoading，避免恢复时闪一下加载态。
  //
  // 重取过程本身会发请求，而每个成功的请求都会再触发一次 core-service-ok，
  // 因此需要 resyncInFlight 兜住重入，避免一次恢复引发多轮重复拉取。
  resyncCore: async () => {
    if (resyncInFlight) return;
    resyncInFlight = true;
    try {
      const [appConfig, deviceStatus, debugInfo] = await Promise.all([
        configService.getConfig(),
        deviceService.getStatus() as Promise<DeviceStatusPayload>,
        apiService.getDebugInfo().catch(() => null),
      ]);
      const coreServiceError = deviceStatus.error ? getCoreServiceErrorMessage(deviceStatus.error) : null;

      set({
        config: appConfig,
        isConnected: deviceStatus.connected || false,
        deviceProductId: deviceStatus.productId || null,
        deviceModel: deviceStatus.model || null,
        deviceSettings: deviceStatus.deviceSettings || null,
        fanData: deviceStatus.currentData || null,
        legionFnQSupported: debugInfo?.legionFnQSupported === true,
        coreServiceError,
        coreServiceErrorRaw: deviceStatus.error || null,
        error: coreServiceError,
      });

      get().handleTemperaturePayload(deviceStatus.temperature || null);
    } catch (error) {
      console.error('核心状态重同步失败:', error);
    } finally {
      resyncInFlight = false;
    }
  },

  connectDevice: async () => {
    try {
      const success = await deviceService.connect();
      if (success) {
        const status = await deviceService.getStatus().catch(() => null);
        const coreServiceError = status?.error ? getCoreServiceErrorMessage(status.error) : null;
        set({
          isConnected: true,
          deviceSettings: status?.deviceSettings || null,
          deviceProductId: status?.productId || get().deviceProductId,
          deviceModel: status?.model || get().deviceModel,
          coreServiceError,
          coreServiceErrorRaw: status?.error || null,
          error: coreServiceError,
        });
      }
    } catch (error) {
      console.error('连接失败:', error);
      set({ error: i18n.t('store.errors.connectDevice') });
    }
  },

  disconnectDevice: async () => {
    try {
      await deviceService.disconnect();
      set({ isConnected: false, deviceProductId: null, deviceModel: null, deviceSettings: null, fanData: null });
    } catch (error) {
      console.error('断开连接失败:', error);
    }
  },

  updateConfig: async (config) => {
    try {
      await configService.updateConfig(config);
      set({ config, error: null });
    } catch (error) {
      console.error('配置更新失败:', error);
      set({ error: i18n.t('store.errors.saveConfig') });
      toast.error(i18n.t('store.errors.saveConfig'));
      throw error;
    }
  },

  startEventListeners: () => {
    if (typeof window === 'undefined' || !(window as any).runtime?.EventsOnMultiple) {
      return () => {};
    }
    const unsubscribers: Array<() => void> = [];
    let telemetryTimer: number | null = null;
    let pendingFanData: types.FanData | null | undefined;
    let pendingTemperature: types.TemperatureData | null | undefined;
    const flushTelemetry = () => {
      telemetryTimer = null;
      if (pendingFanData !== undefined) {
        set({ fanData: pendingFanData });
        pendingFanData = undefined;
      }
      if (pendingTemperature !== undefined) {
        const data = pendingTemperature;
        pendingTemperature = undefined;
        get().handleTemperaturePayload(data);
        get().appendSessionHistoryPoint(data);
      }
    };
    const scheduleTelemetryFlush = () => {
      if (telemetryTimer !== null) return;
      telemetryTimer = window.setTimeout(flushTelemetry, document.hidden ? 1000 : 200);
    };

    unsubscribers.push(
      apiService.onCoreServiceError((message) => {
        const coreServiceError = getCoreServiceErrorMessage(message);
        set({
          coreServiceError,
          coreServiceErrorRaw: message || null,
          error: coreServiceError,
          isConnected: false,
          deviceProductId: null,
          deviceModel: null,
          deviceSettings: null,
          fanData: null,
        });
      })
    );

    unsubscribers.push(
      apiService.onCoreServiceOK(() => {
        // 该事件在每次请求成功时都会到达，只有"从错误态恢复"这一次跳变需要重同步。
        const recovering = get().coreServiceError !== null;
        set((state) => ({
          coreServiceError: null,
          coreServiceErrorRaw: null,
          error: state.coreServiceError && state.error === state.coreServiceError ? null : state.error,
        }));
        if (recovering) {
          void get().resyncCore();
        }
      })
    );

    // 核心侧 IPC 看护协程重连成功后主动推送，用于覆盖"GUI 空闲期间核心重启"
    // 这类没有任何请求在飞、因而不会触发 core-service-ok 跳变的场景。
    unsubscribers.push(
      apiService.onCoreResynced(() => {
        void get().resyncCore();
      })
    );

    unsubscribers.push(
      deviceService.onDeviceConnected((deviceInfo) => {
        console.log('设备已连接:', deviceInfo);
        const info = deviceInfo as { productId?: string; model?: string };
        const settings = (deviceInfo as { deviceSettings?: DeviceSettings | null })?.deviceSettings || null;
        set({
          isConnected: true,
          deviceProductId: info.productId || null,
          deviceModel: info.model || null,
          deviceSettings: settings,
          coreServiceError: null,
          coreServiceErrorRaw: null,
          error: null,
        });
      })
    );

    unsubscribers.push(
      deviceService.onDeviceDisconnected(() => {
        console.log('设备已断开');
        set({ isConnected: false, deviceProductId: null, deviceModel: null, deviceSettings: null, fanData: null });
      })
    );

    // 时间轴标记统一由核心推送：核心常驻后台，界面关着时发生的断连/唤醒也记得下来，
    // 打开界面后连同历史快照一起补齐。前端自己从 IPC 事件推导只能覆盖开着界面的那段。
    unsubscribers.push(
      apiService.onTimelineEvent((event) => {
        get().mergeTimelineEvents([event]);
      })
    );

    unsubscribers.push(
      deviceService.onDeviceSettingsUpdate((settings) => {
        set({ deviceSettings: settings || null });
      })
    );

    unsubscribers.push(
      deviceService.onDeviceError((errorMsg) => {
        console.error('设备错误:', errorMsg);
        set({ error: getLocalizedDeviceError(errorMsg) });
      })
    );

    unsubscribers.push(
      deviceService.onFanDataUpdate((data) => {
        pendingFanData = data;
        scheduleTelemetryFlush();
      })
    );

    unsubscribers.push(
      deviceService.onTemperatureUpdate((data) => {
        pendingTemperature = data;
        scheduleTelemetryFlush();
      })
    );

    unsubscribers.push(
      configService.onConfigUpdate((updatedConfig) => {
        set({ config: updatedConfig });
      })
    );

    unsubscribers.push(
      deviceService.onHotkeyTriggered((payload) => {
        const message = typeof payload?.message === 'string' ? payload.message : '';
        // 核心带了 i18n 键就用它（成功路径都有），否则回退到核心给的中文原文——失败路径
        // 送上来的是 err.Error()，没有对应的键。
        const messageKey = typeof payload?.messageKey === 'string' ? payload.messageKey : '';
        const description = messageKey
          ? i18n.t(messageKey, translateHotkeyParams(payload?.messageParams))
          : message;
        if (!description) return;
        const ok = payload?.success !== false;
        if (ok) {
          toast.success(i18n.t('store.hotkey.successTitle'), { description, duration: 2600 });
        } else {
          toast.error(i18n.t('store.hotkey.failureTitle'), { description, duration: 3200 });
        }
      })
    );

    unsubscribers.push(
      deviceService.onLegionPowerModeUpdate((payload) => {
        const mode = typeof payload?.mode === 'string' ? payload.mode : '';
        if (!mode) return;
        const modeLabel: Record<string, string> = {
          Quiet: i18n.t('store.legionModes.Quiet'),
          Balance: i18n.t('store.legionModes.Balance'),
          Performance: i18n.t('store.legionModes.Performance'),
          Extreme: i18n.t('store.legionModes.Extreme'),
          GodMode: i18n.t('store.legionModes.GodMode'),
        };
        toast.success(i18n.t('store.legionFnQ.modeChangedTitle'), {
          description: i18n.t('store.legionFnQ.modeDescription', { mode: modeLabel[mode] || mode }),
          duration: 2600,
        });
      })
    );

    unsubscribers.push(
      apiService.onLegionFnQSupportUpdate((payload) => {
        set({ legionFnQSupported: payload?.supported === true });
      })
    );

    return () => {
      if (telemetryTimer !== null) window.clearTimeout(telemetryTimer);
      unsubscribers.forEach((unsubscribe) => unsubscribe());
    };
  },
}));
