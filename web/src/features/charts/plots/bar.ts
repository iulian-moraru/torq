import ChartCanvas from "../chartCanvas";
import { AbstractPlot, basePlotConfig, drawConfig } from "./abstract";
import { addHours } from "date-fns";
import * as d3 from "d3";
import { Selection } from "d3";

type barsConfig = basePlotConfig & {
  key: string; // The key used to fetch data
  barGap: number; // The gap between each bar
  barColor: string; // The color of the bar
  barHoverColor: string;
  legendLabel?: string;
  labels?: boolean;
  textColor: string;
  textHoverColor: string;
  rightAxis: boolean;
};
type barInputConfig = Partial<barsConfig> & Required<Pick<barsConfig, "id" | "key">>;

export class BarPlot extends AbstractPlot {
  config: barsConfig;
  legend: Selection<HTMLDivElement, Record<string, never>, HTMLElement, any>;
  legendTextBox: Selection<HTMLDivElement, Record<string, never>, HTMLElement, any>;
  legendColorBox: Selection<HTMLDivElement, Record<string, never>, HTMLElement, any>;
  /**
   * Plots bars on a chart canvas. To use it add it to the plots map on the Chart instance.
   *
   * @param chart - The Chart instance where BarPlot will be plotted on
   * @param config - Plot config, only required attributes are key and ID
   */
  constructor(chart: ChartCanvas, config: barInputConfig) {
    super(chart, config);

    this.config = {
      rightAxis: false,
      barGap: 0.1,
      barColor: "#B6DCFF",
      barHoverColor: "#9DD0FF",
      textColor: "#8198A3",
      textHoverColor: "#3A463C",
      ...config,
    };

    this.legend = this.chart.legendContainer
      .append("div")
      .attr("class", "legendContent")
      .attr("id", `${this.config.id}`);

    this.legendColorBox = this.legend
      .append("div")
      .attr("class", "legendColorBox")
      .attr("style", `width: 12px; height: 12px; background: ${this.config.barColor};`);

    this.legend
      .append("div")
      .attr("class", "legendLabelBox")
      .text((this.config.legendLabel || "") + ": ");

    this.legendTextBox = this.legend.append("div").attr("class", "legendTextBox");
  }

  /**
   * xPoint returns the starting location for the bar on the xScale in pixels
   *
   * @param xValue the data point on the xScale that you want to convert to a pixel location on the chart.
   */
  xPoint(xValue: number): number {
    return (this.chart.config.xScale(xValue) || 0) - this.barWidth() / 2;
  }

  height(dataPoint: number): number {
    const yScale = this.config.rightAxis ? this.chart.config.rightYScale : this.chart.config.yScale;
    return yScale(dataPoint);
  }

  barWidth(): number {
    return this.chart.tickWidth();
  }

  drawBar(context: CanvasRenderingContext2D, dataPoint: any, fillColor: string) {
    context.fillStyle = fillColor;
    context.strokeStyle = fillColor;

    // Draw the bar rectangle
    context.fillRect(
      this.xPoint(dataPoint.date) + (this.barWidth() * this.config.barGap) / 2,
      this.yPoint(dataPoint[this.config.key]),
      this.barWidth() * (1 - this.config.barGap),
      this.height(-dataPoint[this.config.key])
    );
  }

  /**
   * Draw draws the bars on the Chart instance based on the configuration provided.
   */
  draw(drawConfig?: drawConfig) {
    this.chart.data.forEach((data, i) => {
      this.chart.context.fillStyle = this.config.barColor;

      let xIndex = -1;
      if (
        drawConfig?.xPosition &&
        addHours(this.chart.config.xScale.invert(drawConfig.xPosition), 12) > data.date &&
        addHours(this.chart.config.xScale.invert(drawConfig.xPosition), 12) < addHours(data.date, 24)
      ) {
        xIndex = i;
      }

      const hoversOverDataPoint = xIndex === i;

      let barColor = this.config.barColor;
      if (hoversOverDataPoint) {
        barColor = this.config.barHoverColor;
      }
      this.drawBar(this.chart.context, this.chart.data[i], barColor);

      // Create the interaction color and
      const interactionColor = this.chart.genColor();
      // this.chart.figures.set(interactionColor, {
      //   plot: this,
      //   interactionConfig: { hoverIndex: i },
      // });

      this.chart.interactionContext.fillStyle = interactionColor;
      this.chart.interactionContext.fillRect(
        Math.round(this.xPoint(this.chart.data[i].date)),
        Math.round(this.yPoint(this.chart.data[i][this.config.key])),
        Math.round(this.barWidth()),
        Math.round(this.height(-this.chart.data[i][this.config.key]))
      );
    });
    this.chart.data.forEach((data, i) => {
      let xIndex = -1;
      if (
        drawConfig?.xPosition &&
        addHours(this.chart.config.xScale.invert(drawConfig.xPosition), 12) > data.date &&
        addHours(this.chart.config.xScale.invert(drawConfig.xPosition), 12) < addHours(data.date, 24)
      ) {
        xIndex = i;
      }
      const hoversOverDataPoint = xIndex === i;
      let textColor = this.config.textColor;
      if (hoversOverDataPoint) {
        textColor = this.config.textHoverColor;
      }

      if (this.config.labels) {
        this.chart.context.font = "12px Inter";
        this.chart.context.textAlign = "center";
        this.chart.context.textBaseline = "middle";
        this.chart.context.fillStyle = textColor;
        this.chart.context.fillText(
          d3.format(",")(this.chart.data[i][this.config.key]),
          this.xPoint(this.chart.data[i].date) + this.barWidth() / 2,
          this.yPoint(this.chart.data[i][this.config.key]) - 15
        );
      }
      let hoverIndex: number;
      switch (drawConfig?.xIndex) {
        case undefined:
          hoverIndex = this.chart.data.length - 1;
          break;
        case 0:
          hoverIndex = 0;
          break;
        default:
          hoverIndex = drawConfig?.xIndex || 0;
      }
      const legendText = this.chart.data[hoverIndex][this.config.key];

      this.legendTextBox.text(d3.format(",")(legendText));
    });
  }
}
