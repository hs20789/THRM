'use client';

import { useTranslation } from 'react-i18next';
import { toast, type ExternalToast } from 'sonner';
import { formatDisplayMessage, type DisplayMessageInput } from './display-localization';

export function DisplayText({ message }: { message: DisplayMessageInput }) {
  const { t } = useTranslation();
  return <span className="whitespace-pre-wrap">{formatDisplayMessage(message, t)}</span>;
}

type DisplayToastOptions = Omit<ExternalToast, 'description'> & { description?: DisplayMessageInput };
const show = (kind: 'success' | 'error' | 'warning' | 'info', message: DisplayMessageInput, options?: DisplayToastOptions) =>
  toast[kind](<DisplayText message={message} />, {
    ...options,
    description: options?.description == null ? undefined : <DisplayText message={options.description} />,
  });

// React content subscribes to locale changes even while an existing toast is on screen.
export const displayToast = {
  success: (message: DisplayMessageInput, options?: DisplayToastOptions) => show('success', message, options),
  error: (message: DisplayMessageInput, options?: DisplayToastOptions) => show('error', message, options),
  warning: (message: DisplayMessageInput, options?: DisplayToastOptions) => show('warning', message, options),
  info: (message: DisplayMessageInput, options?: DisplayToastOptions) => show('info', message, options),
};
