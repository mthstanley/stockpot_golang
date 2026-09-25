export const setEmptyOrStr = (str: string): string | null =>
  str === "" ? null : str;

// eslint-disable-next-line  @typescript-eslint/no-explicit-any
type NotObject = Exclude<any, object>;
interface NestedObjectType {
  [key: string]: NotObject | NestedObjectType;
}

export const stripEmpty = <T extends NestedObjectType>(obj: T): T => {
  return Object.entries(obj)
    .map(([k, v]) => [
      k,
      v && typeof v === "object" && !Array.isArray(v) ? stripEmpty(v) : v,
    ])
    .reduce((a, [k, v]) => {
      if (v != null && !Number.isNaN(v)) {
        if (Array.isArray(v)) {
          a[k] = v
            .map((e) =>
              e && typeof e === "object" && !Array.isArray(e)
                ? stripEmpty(e)
                : e,
            )
            .filter((e) => !(e == null));
        } else {
          a[k] = v;
        }
      }
      return a;
    }, {} as NestedObjectType) as T;
};
