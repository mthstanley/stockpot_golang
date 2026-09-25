const camelToSnakeCase = (str: string): string =>
  str.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`);

const snakeCaseToCamel = (str: string): string =>
  str.replace(/([_][a-z])/gi, (match) => match.toUpperCase().replace("_", ""));

// eslint-disable-next-line  @typescript-eslint/no-explicit-any
const isObject = (o: any): boolean =>
  o === Object(o) && !Array.isArray(o) && typeof o !== "function";

// eslint-disable-next-line  @typescript-eslint/no-explicit-any
const mapKeys = (o: any, func: (s: string) => string): any => {
  if (isObject(o)) {
    // eslint-disable-next-line  @typescript-eslint/no-explicit-any
    const n: Record<string, any> = {};

    Object.keys(o).forEach((k) => {
      n[func(k)] = mapKeys(o[k], func);
    });

    return n;
  } else if (Array.isArray(o)) {
    return o.map((i) => {
      return mapKeys(i, func);
    });
  }

  return o;
};

// eslint-disable-next-line  @typescript-eslint/no-explicit-any
export const camelKeys = (o: any): any => mapKeys(o, snakeCaseToCamel);

// eslint-disable-next-line  @typescript-eslint/no-explicit-any
export const snakeKeys = (o: any): any => mapKeys(o, camelToSnakeCase);
