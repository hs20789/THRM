import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createNativeLocaleSync } from '../src/app/lib/native-locale-sync.ts';

const tick = () => new Promise(resolve => setImmediate(resolve));

test('does not send default Chinese before the saved locale is ready; resends after Core reconnect', async () => {
  const calls = [];
  const sync = createNativeLocaleSync(async locale => { calls.push(locale); }, assert.fail);
  sync.resync(); sync.retry();
  await tick();
  assert.deepEqual(calls, []);
  sync.setLocale('ko-KR');
  await tick();
  sync.retry();
  assert.deepEqual(calls, ['ko-KR']);
  sync.resync();
  await tick();
  assert.deepEqual(calls, ['ko-KR', 'ko-KR']);
  sync.dispose();
});

test('serializes language switches and retains the latest selection', async () => {
  const calls = [], complete = [];
  const sync = createNativeLocaleSync(locale => { calls.push(locale); return new Promise(resolve => complete.push(resolve)); }, assert.fail);
  sync.setLocale('zh-CN');
  sync.setLocale('ko-KR');
  sync.setLocale('en-US');
  assert.deepEqual(calls, ['zh-CN']);
  complete.shift()(); await tick();
  assert.deepEqual(calls, ['zh-CN', 'en-US']);
  complete.shift()(); await tick();
  sync.setLocale('ja-JP');
  assert.deepEqual(calls, ['zh-CN', 'en-US', 'ja-JP']);
  complete.shift()(); await tick();
  sync.dispose();
});

test('failed synchronization retries on recovery without a feedback loop', async () => {
  let attempts = 0;
  const errors = [];
  let sync;
  sync = createNativeLocaleSync(async () => {
    attempts++;
    if (attempts === 1) throw new Error('Core disconnected');
    sync.retry(); // SetUILocale itself emits core-service-ok before resolving.
  }, error => errors.push(error));
  sync.setLocale('ko-KR'); await tick();
  assert.equal(attempts, 1);
  assert.equal(errors.length, 1);
  sync.retry(); await tick();
  sync.retry(); await tick();
  assert.equal(attempts, 2);
  sync.dispose(); sync.resync();
  await tick(); assert.equal(attempts, 2);
});
