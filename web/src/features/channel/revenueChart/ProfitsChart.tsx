// https://www.pluralsight.com/guides/using-d3.js-inside-a-react-app
import { useD3 } from "../../charts/useD3";
import { useEffect } from "react";
import { Selection } from "d3";
import ChartCanvas from "../../charts/chartCanvas";
import "../../charts/chart.scss";
import { BarPlot, LinePlot } from "../../charts/charts";
import { selectProfitChartKey } from "../channelSlice";
import { useAppSelector } from "../../../store/hooks";
import { useGetSettingsQuery } from "../../../apiSlice";

type ProfitsChart = {
  data: any[];
  dashboard?: boolean;
  from: string;
  to: string;
};

function ProfitsChart({ data, dashboard, to, from }: ProfitsChart) {
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
      if (dashboard) {
        chart = new ChartCanvas(container, data, {
          from: new Date(from),
          to: new Date(to),
          timezone: settings?.data?.preferredTimezone || "UTC",
          yScaleKey: profitKey.value + "_out",
          rightYScaleKey: profitKey.value + "_out",
          rightYAxisKeys: [profitKey.value + "_out"],
          xAxisPadding: 12,
        });
        chart.plot(BarPlot, {
          id: profitKey.value + "_out",
          key: profitKey.value + "_out",
          legendLabel: profitKey.label + " out",
          barColor: "rgba(133, 196, 255, 0.5)",
          // areaGradient: ["rgba(133, 196, 255, 0.5)", "rgba(87, 211, 205, 0.5)"],
        });
        chart.draw();
      } else {
        chart = new ChartCanvas(container, data, {
          from: new Date(from),
          to: new Date(to),
          timezone: settings?.data?.preferredTimezone || "UTC",
          yScaleKey: profitKey.value + "_total",
          rightYScaleKey: profitKey.value + "_total",
          rightYAxisKeys: [profitKey.value + "_out", profitKey.value + "_in"],
          xAxisPadding: 12,
        });
        chart.plot(BarPlot, {
          id: profitKey.value + "_total",
          key: profitKey.value + "_total",
          legendLabel: profitKey.label + " total",
          // areaGradient: ["rgba(133, 196, 255, 0.5)", "rgba(87, 211, 205, 0.5)"],
          barColor: "rgba(133, 196, 255, 0.5)",
        });
        chart.plot(LinePlot, {
          id: profitKey.value + "_out",
          key: profitKey.value + "_out",
          legendLabel: profitKey.label + " out",
          lineColor: "#BA93FA",
        });
        chart.plot(LinePlot, {
          id: profitKey.value + "_in",
          key: profitKey.value + "_in",
          legendLabel: profitKey.label + " in",
          lineColor: "#FAAE93",
        });
        chart.draw();
      }

      setInterval(navCheck(container), 200);
    },
    [data, data ? data[0].date : "", data ? data[data.length - 1].date : "", profitKey, settings]
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

export default ProfitsChart;
