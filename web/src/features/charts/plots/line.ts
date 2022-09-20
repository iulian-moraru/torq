import ChartCanvas from "../chartCanvas";
import { AbstractPlot, basePlotConfig, drawConfig } from "./abstract";
import * as d3 from "d3";
import { Selection } from "d3";

type linePlotConfig = basePlotConfig & {
  lineColor: string;
  lineWidth: number;
  labels?: boolean;
  legendLabel?: string;
  rightAxis: boolean;
  curveFunction?: any;
};

type linePlotConfigInit = Partial<linePlotConfig> & basePlotConfig;

export class LinePlot extends AbstractPlot {
  config: linePlotConfig;
  legend: Selection<HTMLDivElement, Record<string, never>, HTMLElement, any>;
  legendTextBox: Selection<HTMLDivElement, Record<string, never>, HTMLElement, any>;
  legendColorBox: Selection<HTMLDivElement, Record<string, never>, HTMLElement, any>;

  constructor(chart: ChartCanvas, config: linePlotConfigInit) {
    super(chart, config);

    this.config = {
      lineColor: "#85C4FF",
      lineWidth: 1.7,
      rightAxis: false,
      ...config,
    };

    this.legend = this.chart.legendContainer
      .append("div")
      .attr("class", "legendContent")
      .attr("id", `${this.config.id}`);

    this.legendColorBox = this.legend.append("div");

    if (this.config.legendLabel) {
      this.legendColorBox
        .attr("class", "legendColorBox")
        .attr("style", `width: 12px; height: 12px; background: ${this.config.lineColor};`);
      this.legend
        .append("div")
        .attr("class", "legendLabelBox")
        .text((this.config.legendLabel || "") + ": ");
    }
    this.legendTextBox = this.legend.append("div").attr("class", "legendTextBox");
  }

  height(dataPoint: number): number {
    const yScale = this.config.rightAxis ? this.chart.config.rightYScale : this.chart.config.yScale;
    return yScale(dataPoint);
  }

  draw(drawConfig?: drawConfig) {
    const yScale = this.config.rightAxis ? this.chart.config.rightYScale : this.chart.config.yScale;
    const line = d3
      .line()
      .x((_, i): number => {
        return this.chart.config.xScale(this.chart.data[i].date) || 0;
      })
      .y((_, i): number => {
        return yScale(this.chart.data[i][this.config.key]) || 0;
      })
      .context(this.chart.context);

    if (this.config.curveFunction) {
      line.curve(this.config.curveFunction);
    }

    this.chart.context.strokeStyle = this.config.lineColor;

    this.chart.context.beginPath();
    line(this.chart.data);

    this.chart.context.lineWidth = this.config.lineWidth;
    this.chart.context.stroke();

    if (this.config.labels) {
      this.chart.data.forEach((d, _) => {
        this.chart.context.font = "12px Inter";
        this.chart.context.textAlign = "center";
        this.chart.context.textBaseline = "middle";
        this.chart.context.fillStyle = "#3A463C";
        this.chart.context.fillText(
          d3.format(",")(d[this.config.key]),
          this.xPoint(d.date),
          this.yPoint(d[this.config.key]) - 15
        );
      });
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
    if (this.config.legendLabel) {
      const legendText = this.chart.data[hoverIndex][this.config.key];
      this.legendTextBox.text(d3.format(",")(legendText));
    }
  }
}
