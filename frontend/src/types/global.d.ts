export {};

declare global {
  interface Window {
    env?: {
      DISABLE_USER_SIGNUP: boolean;
    };
  }
}
