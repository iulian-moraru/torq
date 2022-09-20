// https://www.pluralsight.com/guides/using-d3.js-inside-a-react-app
import React from "react";
import * as d3 from "d3";
import { BaseType } from "d3";

// eslint-disable-next-line @typescript-eslint/ban-types
export const useD3 = (renderChartFn: Function, dependencies: any) => {
  const ref = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    renderChartFn(d3.select(ref.current as unknown as BaseType));
    return () => {
      return;
    };
  }, dependencies);

  return ref;
};
