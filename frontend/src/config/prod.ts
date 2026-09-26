import AppConfig from "./types";

const config: AppConfig = {
  apiBaseUrl: new URL(import.meta.env.VITE_API_URL),
};

export default config;
