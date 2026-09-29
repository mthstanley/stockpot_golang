import AppConfig from "./types";

const config: AppConfig = {
  disableUserSignup: window.env?.DISABLE_USER_SIGNUP ?? false,
};

export { config };
