// The Android shell (WebBridge.kt). The inline script in the head of
// index.html reads the safe area from it at parse time; the settings page
// below offers what only the app can do when it is there.
interface Window {
  vfoxAndroid?: {
    insets(): string;
    server(): string;
    pairNew(): void;
    disconnect(): void;
  };
  vfoxInsets?: () => void;
}
