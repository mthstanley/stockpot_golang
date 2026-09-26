import local from "./local";
import prod from "./prod";
import AppConfig from "./types";

const config: AppConfig = { local, prod }[import.meta.env.VITE_APP_ENV || "local"]!;

export { config };
