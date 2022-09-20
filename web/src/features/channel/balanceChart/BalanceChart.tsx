// https://www.pluralsight.com/guides/using-d3.js-inside-a-react-app
import { useD3 } from "../../charts/useD3";
import * as d3 from "d3";
import { useEffect } from "react";
import { NumberValue, Selection } from "d3";
import ChartCanvas from "../../charts/chartCanvas";
import "../../charts/chart.scss";
import { AreaPlot } from "../../charts/charts";
import { selectProfitChartKey } from "../channelSlice";
import { useAppSelector } from "../../../store/hooks";
import clone from "../../../clone";
import { useGetSettingsQuery } from "../../../apiSlice";

type BalanceChart = {
  data: any[];
  totalCapacity: number;
  from: string;
  to: string;
};

function BalanceChart({ data, totalCapacity, from, to }: BalanceChart) {
  let chart: ChartCanvas;
  let currentSize: [number | undefined, number | undefined] = [undefined, undefined];
  const profitKey = useAppSelector(selectProfitChartKey);
  const settings = useGetSettingsQuery();

  // Check and update the chart size if the navigation changes the container size
  const navCheck = (container: Selection<HTMLDivElement, Record<string, never>, HTMLElement, any>) => {
    return () => {
      const boundingBox = container?.node()?.getBoundingClientRect();
      if (currentSize[0] !== boundingBox?.width || currentSize[1] !== boundingBox?.height) {
        chart.resizeChart();
        chart.draw();
        currentSize = [boundingBox?.width, boundingBox?.height];
      }
    };
  };

  // TODO: Change this so that we can update the data without redrawing the entire chart
  const ref = useD3(
    (container: Selection<HTMLDivElement, Record<string, never>, HTMLElement, any>) => {
      let paddedData: any[] = [];

      if (data?.length > 0) {
        paddedData = clone(data);
        paddedData.unshift({
          capacity_diff: 0,
          date: from,
          inbound_capacity: data[0].inbound_capacity,
          outbound_capacity: data[0].outbound_capacity,
        });
        paddedData.push({
          capacity_diff: 0,
          date: to,
          inbound_capacity: data[Math.max(data.length - 1, 0)].inbound_capacity,
          outbound_capacity: data[Math.max(data.length - 1, 0)].outbound_capacity,
        });
      }

      chart = new ChartCanvas(container, paddedData, {
        from: new Date(from),
        to: new Date(to),
        timezone: settings?.data?.preferredTimezone || "UTC",
        yScaleKey: "outbound_capacity",
        rightYScaleKey: "outbound_capacity",
        rightYAxisKeys: ["outbound_capacity"],
        yAxisMaxOverride: totalCapacity,
        rightYAxisMaxOverride: totalCapacity,
        xAxisLabelFormatter: d3.timeFormat("%d %b - %H:%M") as (domainValue: NumberValue) => string,
      });

      chart.plot(AreaPlot, {
        id: "outbound_capacity",
        key: "outbound_capacity",
        legendLabel: "Outbound capacity",
        curveFunction: d3.curveStepAfter,
        areaGradient: ["rgba(133, 196, 255, 0.5)", "rgba(87, 211, 205, 0.6)"],
        // areaColor: "#FAAE93",
      });
      chart.draw();

      setInterval(navCheck(container), 200);
    },
    [data, data ? data[0].date : "", data ? data[data.length - 1].date : "", profitKey]
  );

  useEffect(() => {
    return () => {
      if (chart) {
        chart.removeResizeListener();
      }
    };
  }, [data]);

  return <div ref={ref} className={"chart-ref"} />;
}

export default BalanceChart;
