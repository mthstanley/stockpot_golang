import AppConfig from "./types";

const config: AppConfig = {
  disableUserSignup: window.env?.DISABLE_USER_SIGNUP ?? false,
  apiHost: import.meta.env.VITE_API_URL ?? "/api",
};

export { config };
