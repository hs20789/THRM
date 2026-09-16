import { useEffect, useRef } from 'react';
import { useLocale } from '../lib/i18n';
import { createNativeLocaleSync } from '../lib/native-locale-sync';
import { apiService } from '../services/api';

export function useNativeLocaleSync() {
  const { locale, localeReady } = useLocale();
  const sync = useRef<ReturnType<typeof createNativeLocaleSync> | null>(null);

  useEffect(() => {
    if (!(window as any).go?.main?.App) return;
    const controller = createNativeLocaleSync(
      (value) => apiService.setUILocale(value),
      (error) => console.warn('Native UI locale sync failed; will retry', error),
    );
    sync.current = controller;
    const stopResynced = apiService.onCoreResynced(() => controller.resync());
    const stopOK = apiService.onCoreServiceOK(() => controller.retry());
    const timer = window.setInterval(() => controller.retry(), 5000);
    return () => {
      controller.dispose();
      sync.current = null;
      stopResynced();
      stopOK();
      window.clearInterval(timer);
    };
  }, []);

  useEffect(() => {
    if (localeReady) sync.current?.setLocale(locale);
  }, [locale, localeReady]);
}
