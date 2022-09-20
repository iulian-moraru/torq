// https://www.pluralsight.com/guides/using-d3.js-inside-a-react-app
import { useD3 } from "../../charts/useD3";
import { useEffect } from "react";
import { Selection } from "d3";
import { ChartCanvas, EventsPlot, LinePlot, BarPlot } from "../../charts/charts";
import "../../charts/chart.scss";
import { useAppSelector } from "../../../store/hooks";
import { selectEventChartKey } from "../channelSlice";
import { useGetSettingsQuery } from "../../../apiSlice";

type EventsChart = {
  data: any[];
  events: any[];
  selectedEventTypes: Map<string, boolean>;
  from: string;
  to: string;
};

function EventsChart({ data, events, selectedEventTypes, from, to }: EventsChart) {
  let chart: ChartCanvas;
  let currentSize: [number | undefined, number | undefined] = [undefined, undefined];
  const eventKey = useAppSelector(selectEventChartKey);
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
      chart = new ChartCanvas(container, data, {
        from: new Date(from),
        to: new Date(to),
        timezone: settings?.data?.preferredTimezone || "UTC",
        yScaleKey: eventKey.value + "_total",
        rightYScaleKey: eventKey.value + "_total",
        rightYAxisKeys: [eventKey.value + "_out", eventKey.value + "_in", eventKey.value + "_total"],
        xAxisPadding: 12,
      });
      chart.plot(BarPlot, {
        id: eventKey.value + "_total",
        key: eventKey.value + "_total",
        legendLabel: eventKey.label + " Total",
        barColor: "rgba(133, 196, 255, 0.5)",
        // areaGradient: ["rgba(133, 196, 255, 0.5)", "rgba(87, 211, 205, 0.5)"],
      });
      chart.plot(LinePlot, {
        id: eventKey.value + "_out",
        key: eventKey.value + "_out",
        legendLabel: eventKey.label + " Out",
        lineColor: "#BA93FA",
        // rightAxis: true,
      });
      chart.plot(LinePlot, {
        id: eventKey.value + "_in",
        key: eventKey.value + "_in",
        legendLabel: eventKey.label + " In",
        lineColor: "#FAAE93",
      });
      const filteredEvents =
        events?.filter((d) => {
          return selectedEventTypes.get(d.type); // selectedEventTypes
        }) || [];
      chart.plot(EventsPlot, { id: "events", key: "events", events: filteredEvents });
      chart.draw();
      setInterval(navCheck(container), 200);
    },
    [data, eventKey, data ? data[0].date : "", data ? data[data.length - 1].date : "", selectedEventTypes, settings]
  );

  useEffect(() => {
    return () => {
      if (chart) {
        chart.removeResizeListener();
      }
    };
  }, [data, data ? data[0].date : ""]);

  return <div ref={ref} className={"testing"} />;
}

export default EventsChart;
