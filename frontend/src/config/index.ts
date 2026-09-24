import local from "./local";
import AppConfig from "./types";

const config: AppConfig = { local }[import.meta.env.VITE_APP_ENV || "local"]!;

export { config };
