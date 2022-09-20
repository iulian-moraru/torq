import * as d3 from "d3";
import { ScaleLinear, Selection } from "d3";
import clone from "../../clone";

type chartConfig = {
  margin: {
    top: number;
    right: number;
    bottom: number;
    left: number;
  };
  keyOut: keyof Omit<FlowData, "alias" | "chan_id" | "pub_key" | "channel_point">;
  keyIn: keyof Omit<FlowData, "alias" | "chan_id" | "pub_key" | "channel_point">;
  totalInbound: number;
  totalOutbound: number;
  verticalGap: number;
  barWidth: number;
  horizontalGap: number;
  height: number;
  width: number;
  inboundStroke: string;
  outboundStroke: string;
  inboundFill: string;
  outboundFill: string;
  yScale: ScaleLinear<number, number, never>;
  xScale: ScaleLinear<number, number, number | undefined>;
};

export type FlowData = {
  alias: string;
  chan_id: string;
  pub_key: string;
  channel_point: string;
  amount_out: number;
  revenue_out: number;
  count_out: number;
  amount_in: number;
  revenue_in: number;
  count_in: number;
};

class FlowChartCanvas {
  config: chartConfig = {
    margin: {
      top: 0,
      right: 0,
      bottom: 0,
      left: 0,
    },
    keyOut: "amount_out",
    keyIn: "amount_in",
    totalInbound: 0,
    totalOutbound: 0,
    verticalGap: 5,
    barWidth: 10,
    horizontalGap: 7.5,
    height: 200,
    width: 500,
    inboundStroke: "#F3F9FF",
    outboundStroke: "#ECFAF8",
    inboundFill: "#C2E2FF",
    outboundFill: "#ABE9E6",
    yScale: d3.scaleLinear([0, 200]),
    xScale: d3.scaleLinear([0, 500]),
  };

  mouseOver?: { index: number; outbound: boolean };

  dataRaw: Array<FlowData> = [];
  data: Array<FlowData> = [];

  container: Selection<HTMLDivElement, Record<string, never>, HTMLElement, any>;
  chartContainer: Selection<HTMLDivElement, Record<string, never>, HTMLElement, any>;

  labelsContainer: Selection<HTMLDivElement, Record<string, never>, HTMLElement, any>;

  canvas: Selection<HTMLCanvasElement, Record<string, never>, HTMLElement, any>;
  interactionLayer: Selection<HTMLCanvasElement, Record<string, never>, HTMLElement, any>;

  context: CanvasRenderingContext2D;
  interactionContext: CanvasRenderingContext2D;

  constructor(
    container: Selection<HTMLDivElement, Record<string, never>, HTMLElement, any>,
    data: Array<FlowData>,
    config: Partial<chartConfig>
  ) {
    if (container == undefined) {
      throw new Error("The chart container can't be null");
    }

    this.dataRaw = data;

    this.config = { ...this.config, ...config };

    this.container = container.attr("style", "position: relative; height: 100%;").html(null);

    // Configure the chart width and height based on the container
    this.config.width = this.getWidth();
    this.config.height = this.getHeight();

    this.config.totalInbound = data
      .map((d) => d[this.config.keyIn] as number)
      .reduce((partialSum, a) => partialSum + a, 0);
    this.config.totalOutbound = data
      .map((d) => d[this.config.keyOut] as number)
      .reduce((partialSum, a) => partialSum + a, 0);
    const threshold = Math.max(this.config.totalOutbound, this.config.totalInbound) * 0.02;

    // Filters out small channels
    this.data = clone(this.dataRaw);

    const otherChannelsOut: FlowData = {
      alias: "",
      chan_id: "",
      channel_point: "",
      pub_key: "",
      amount_in: 0,
      revenue_in: 0,
      count_in: 0,
      amount_out: 0,
      revenue_out: 0,
      count_out: 0,
    };
    let otherChanOutCount = 0;
    this.data.forEach((d, _) => {
      if ((d[this.config.keyOut] as number) && (d[this.config.keyOut] as number) < threshold) {
        otherChannelsOut.amount_out += d["amount_out"];
        otherChannelsOut.revenue_out += d["revenue_out"];
        otherChannelsOut.count_out += d["count_out"];
        d["amount_out"] = 0;
        d["revenue_out"] = 0;
        d["count_out"] = 0;
        otherChanOutCount++;
      }
    });
    otherChannelsOut.alias = `(${otherChanOutCount} small channels)`;
    this.data.push(otherChannelsOut);

    const otherChannelsIn: FlowData = {
      alias: "",
      chan_id: "",
      channel_point: "",
      pub_key: "",
      amount_in: 0,
      revenue_in: 0,
      count_in: 0,
      amount_out: 0,
      revenue_out: 0,
      count_out: 0,
    };
    let otherChanInCount = 0;
    this.data.forEach((d, _) => {
      if ((d[this.config.keyIn] as number) && (d[this.config.keyIn] as number) < threshold) {
        otherChannelsIn.amount_in += d["amount_in"];
        otherChannelsIn.revenue_in += d["revenue_in"];
        otherChannelsIn.count_in += d["count_in"];
        d["amount_in"] = 0;
        d["revenue_in"] = 0;
        d["count_in"] = 0;
        otherChanInCount++;
      }
    });
    otherChannelsIn.alias = `(${otherChanInCount} small channels)`;
    this.data.push(otherChannelsIn);

    this.config.xScale = d3
      .scaleLinear()
      .range([0, this.config.width - this.config.margin.right - this.config.margin.left]);

    this.config.yScale = d3
      .scaleLinear()
      .range([0, this.config.height - this.config.margin.top - this.config.margin.bottom])
      .domain([0, Math.max(this.config.totalInbound, this.config.totalOutbound)]);

    const longestIndex = Math.max(
      this.data.filter((d) => (d[this.config.keyOut] as number) !== 0).length,
      this.data.filter((d) => (d[this.config.keyIn] as number) !== 0).length
    );

    this.config.yScale.domain([
      0,
      Math.max(this.config.totalInbound, this.config.totalOutbound) * 1.02 +
        this.config.yScale.invert(this.config.verticalGap * longestIndex),
    ]);

    this.chartContainer = this.container
      .append("div")
      .attr("class", "chartContainer")
      .attr("width", this.config.width - this.config.margin.left - this.config.margin.right)
      .attr("height", this.config.height - this.config.margin.top - this.config.margin.bottom)
      .attr("style", `position: absolute; left: ${this.config.margin.left}px; top: ${this.config.margin.top}px;`);

    this.canvas = this.chartContainer
      .append("canvas")
      .attr("width", this.config.xScale.range()[1])
      .attr("height", this.config.yScale.range()[1])
      .attr("style", `position: absolute; left: 0; top: 0px;`);

    this.interactionLayer = this.chartContainer
      .append("canvas")
      .attr("width", this.config.xScale.range()[1])
      .attr("height", this.config.yScale.range()[1])
      .attr(
        "style",
        "position: absolute; left: 0; top: 0px; display: none;" // display: none;
      );

    this.context = this.canvas?.node()?.getContext("2d") as CanvasRenderingContext2D;
    this.interactionContext = this.interactionLayer?.node()?.getContext("2d") as CanvasRenderingContext2D;
    this.context.imageSmoothingEnabled = false;
    this.interactionContext.imageSmoothingEnabled = false;

    this.labelsContainer = this.container
      .append("div")
      .attr("class", "labelsContainer")
      .attr("width", this.config.xScale.range()[1])
      .attr("height", this.config.yScale.range()[1])
      .attr("style", `position: absolute; left: 0; top: 0px; pointer-events: none;`);

    this.addResizeListener();
    this.addHoverListener();
    this.addMouseOutListener();
  }

  getHeight(): number {
    return this.container?.node()?.getBoundingClientRect().height || this.config.height;
  }

  getWidth(): number {
    return this.container?.node()?.getBoundingClientRect().width || this.config.width;
  }

  addResizeListener() {
    (d3.select(window).node() as EventTarget).addEventListener("resize", (_) => {
      this.resizeChart();
    });
  }

  removeResizeListener() {
    (d3.select(window).node() as EventTarget).removeEventListener("resize", (_) => {
      this.resizeChart();
    });
  }

  clearCanvas() {
    this.context.clearRect(0, 0, this.config.xScale.range()[1], this.config.yScale.range()[1]);
    this.interactionContext.clearRect(0, 0, this.config.xScale.range()[1], this.config.yScale.range()[1]);
  }

  resizeChart() {
    this.config.width = this.getWidth();
    this.config.xScale.range([0, this.config.width - this.config.margin.right - this.config.margin.left]);

    this.config.height = this.getHeight();

    this.config.totalInbound = this.data
      .map((d) => d[this.config.keyIn] as number)
      .reduce((partialSum, a) => partialSum + a, 0);
    this.config.totalOutbound = this.data
      .map((d) => d[this.config.keyOut] as number)
      .reduce((partialSum, a) => partialSum + a, 0);

    const longestIndex = Math.max(
      this.data.filter((d) => (d[this.config.keyOut] as number) !== 0).length,
      this.data.filter((d) => (d[this.config.keyIn] as number) !== 0).length
    );

    this.config.yScale
      .range([0, this.config.height - this.config.margin.top - this.config.margin.bottom])
      .domain([0, Math.max(this.config.totalInbound, this.config.totalOutbound)]);

    this.config.yScale.domain([
      0,
      Math.max(this.config.totalInbound, this.config.totalOutbound) * 1.02 +
        this.config.yScale.invert(this.config.verticalGap * longestIndex),
    ]);

    // Add the domain equivalent distance of 10pixels to the domain

    this.chartContainer
      .attr("width", this.config.width - this.config.margin.left - this.config.margin.right)
      .attr("height", this.config.height - this.config.margin.top - this.config.margin.bottom);

    this.canvas.attr("width", this.config.xScale.range()[1]).attr("height", this.config.yScale.range()[1]);
    this.interactionLayer.attr("width", this.config.xScale.range()[1]).attr("height", this.config.yScale.range()[1]);

    this.labelsContainer.attr("width", this.config.xScale.range()[1]).attr("height", this.config.yScale.range()[1]);

    this.clearCanvas();

    this.draw();
  }

  figures: Map<string, { index: number; outbound: boolean }> = new Map<string, { index: number; outbound: boolean }>();

  getFigure(xLocation: number, yLocation: number): { index: number; outbound: boolean } | undefined {
    const colorData = this.interactionContext.getImageData(xLocation, yLocation, 1, 1).data;

    return this.figures.get("rgb(" + [colorData[0], colorData[1], colorData[2]].join(",") + ")");
  }

  addHoverListener() {
    this.canvas.on("mousemove", (event) => {
      const [xPosition, yPosition] = d3.pointer(event);
      const figure = this.getFigure(xPosition, yPosition);

      if (figure) {
        this.mouseOver = figure;
        this.clearCanvas();
        this.draw();
      } else if (figure !== this.mouseOver) {
        this.mouseOver = undefined;
        this.clearCanvas();
        this.draw();
      }
    });
  }

  addMouseOutListener() {
    this.canvas.on("mouseleave", (_) => {
      this.mouseOver = undefined;
      this.clearCanvas();
      this.draw();
    });
  }

  drawOutboundBars(
    context: CanvasRenderingContext2D,
    value: number,
    outboundSum: number,
    outboundSumPosition: number,
    yOffset: number,
    index: number
  ) {
    // Bars representing the amount of outbound traffic per channel with a gap between each subsequent bar
    context.fillRect(
      0,
      yOffset - (this.config.yScale(outboundSum) + this.config.verticalGap * index),
      this.config.barWidth,
      -this.config.yScale(value) // d.outbound
    );

    // Bar representing the total amount of outbound traffic, same as bars above, but without the gap
    context.fillRect(
      outboundSumPosition,
      yOffset - this.config.yScale(outboundSum),
      this.config.barWidth,
      -this.config.yScale(value)
    );
    context.fill();
  }

  drawOutboundConnectingLines(
    context: CanvasRenderingContext2D,
    value: number,
    outboundSum: number,
    outboundSumPosition: number,
    yOffset: number,
    index: number
  ) {
    let line = d3
      .line()
      .x((d) => d[0])
      .y((d) => d[1])
      .curve(d3.curveBumpX)
      .context(context);

    if (index === 0) {
      line = d3
        .line()
        .x((d) => d[0])
        .y((d) => d[1])
        .context(context);
    }

    context.beginPath();
    line([
      [
        this.config.barWidth,
        yOffset - this.config.yScale(outboundSum) - this.config.yScale(value) / 2 - this.config.verticalGap * index,
      ],
      [outboundSumPosition, yOffset - this.config.yScale(outboundSum) - this.config.yScale(value) / 2],
    ]);
    context.lineWidth = this.config.yScale(value);
    context.stroke();
    context.beginPath();
  }

  drawInboundBars(
    context: CanvasRenderingContext2D,
    value: number,
    inboundSum: number,
    inboundSumPosition: number,
    yOffset: number,
    index: number
  ) {
    // Bars representing the amount of inbound traffic per channel with a gap between each subsequent bar
    context.fillRect(
      inboundSumPosition,
      yOffset - this.config.yScale(inboundSum),
      this.config.barWidth,
      -this.config.yScale(value)
    );

    // Bar representing the total amount of inbound traffic, same as bars above, but without the gap
    context.fillRect(
      this.config.xScale.range()[1] - this.config.barWidth,
      yOffset - (this.config.yScale(inboundSum) + this.config.verticalGap * index),
      this.config.barWidth,
      -this.config.yScale(value)
    );

    context.fill();
  }

  drawInboundConnectingLines(
    context: CanvasRenderingContext2D,
    value: number,
    inboundSum: number,
    inboundSumPosition: number,
    yOffset: number,
    index: number
  ) {
    let line = d3
      .line()
      .x((d) => {
        return d[0];
      })
      .y((d) => {
        return d[1];
      })
      .curve(d3.curveBumpX)
      .context(context);

    if (index === 0) {
      line = d3
        .line()
        .x((d) => d[0])
        .y((d) => d[1])
        .context(context);
    }

    context.beginPath();
    line([
      [
        inboundSumPosition + this.config.barWidth,
        yOffset - this.config.yScale(inboundSum) - this.config.yScale(value) / 2,
      ],
      [
        this.config.xScale.range()[1] - this.config.barWidth,
        yOffset - this.config.yScale(inboundSum) - this.config.yScale(value) / 2 - this.config.verticalGap * index,
      ],
    ]);
    context.lineWidth = this.config.yScale(value);
    context.stroke();
    context.beginPath();
  }

  drawOutboundValueLabels(
    dataPoint: FlowData,
    outboundSum: number,
    outboundSumPosition: number,
    yOffset: number,
    index: number
  ) {
    let hoverClass = "";
    if (index === this.mouseOver?.index && this.mouseOver.outbound === true) {
      hoverClass = "hover";
    }
    this.labelsContainer
      .append("div")
      .attr("class", "flow-outbound-node " + hoverClass)
      .attr(
        "style",
        `top: ${
          yOffset -
          this.config.yScale(outboundSum) -
          this.config.yScale(dataPoint[this.config.keyOut] as number) / 2 -
          this.config.verticalGap * index
        }px; left: ${20}px;`
      )
      .text(dataPoint.alias);

    this.labelsContainer
      .append("div")
      .attr("class", "flow-outbound-value " + hoverClass)
      .attr(
        "style",
        `top: ${
          yOffset - this.config.yScale(outboundSum) - this.config.yScale(dataPoint[this.config.keyOut] as number) / 2
          // middle between the two endpoints
          // (this.config.verticalGap * index) / 2
          // left: ${20 + outboundSumPosition / 2}px; Middle posistion
        }px; left: ${outboundSumPosition - 10}px;`
      )
      .text(d3.format(",")(dataPoint[this.config.keyOut] as number));
  }

  drawInboundValueLabels(
    dataPoint: FlowData,
    inboundSum: number,
    inboundSumPosition: number,
    yOffset: number,
    index: number
  ) {
    let hoverClass = "";
    if (index === this.mouseOver?.index && this.mouseOver.outbound === false) {
      hoverClass = "hover";
    }

    this.labelsContainer
      .append("div")
      .attr("class", "flow-inbound-node " + hoverClass)
      .attr(
        "style",
        `top: ${
          yOffset -
          this.config.yScale(inboundSum) -
          this.config.yScale(dataPoint[this.config.keyIn] as number) / 2 -
          this.config.verticalGap * index
        }px; left: ${this.config.xScale.range()[1] - this.config.barWidth - 10}px;`
      )
      .text(dataPoint.alias);

    this.labelsContainer
      .append("div")
      .attr("class", "flow-inbound-value " + hoverClass)
      .attr(
        "style",
        `top: ${
          yOffset - this.config.yScale(inboundSum) - this.config.yScale(dataPoint[this.config.keyIn] as number) / 2
          // middle between the two endpoints
          // -(this.config.verticalGap * index) / 2
        }px; left: ${inboundSumPosition + this.config.barWidth + 10}px;`
        // center position: ${this.config.xScale.range()[1] - this.config.barWidth - 10 - inboundSumPosition / 2 + 20}px;
      )
      .text(d3.format(",")(dataPoint[this.config.keyIn] as number));
  }

  /**
   * nextCol keeps track of the next unique color used to identify figures (drawn objects) on the canvas.
   */
  nextCol = 1;

  /**
   * @remarks concept taken from https://www.freecodecamp.org/news/d3-and-canvas-in-3-steps-8505c8b27444/
   */
  genColor() {
    const ret = [];
    if (this.nextCol < 16777215) {
      ret.push(this.nextCol & 0xff);
      ret.push((this.nextCol & 0xff00) >> 8);
      ret.push((this.nextCol & 0xff0000) >> 16);
      // Increase by 10 because the drawn figure changes color when it partially touches a pixel
      // when you increase by 10, the drawn color is different enough to prevent confusion between figures
      this.nextCol += 10;
    }
    return "rgb(" + ret.join(",") + ")";
  }

  draw() {
    // Clear the interaction colours
    this.figures.clear();

    // Clear legends
    this.labelsContainer.selectAll("*").remove();

    let inboundSum = 0;
    let outboundSum = 0;
    const yOffset = this.config.yScale.range()[1];
    const outboundSumPosition = this.config.xScale.range()[1] / 2 - this.config.horizontalGap;
    const inboundSumPosition = this.config.xScale.range()[1] / 2 + this.config.horizontalGap;

    let hoverInboundClass = "";
    if (this.mouseOver?.outbound === false) {
      hoverInboundClass = "hover";
    }
    let hoverOutboundClass = "";
    if (this.mouseOver?.outbound === true) {
      hoverOutboundClass = "hover";
    }

    this.data
      .filter((d) => (d[this.config.keyOut] as number) !== 0)
      .sort((a, b) => {
        return (b[this.config.keyOut] as number) - (a[this.config.keyOut] as number);
      })
      .forEach((d, i) => {
        this.context.fillStyle = this.config.outboundFill;
        this.context.strokeStyle = this.config.outboundStroke;

        if (this.mouseOver?.index === i && this.mouseOver?.outbound === true) {
          this.context.fillStyle = "#57D3CD";
          this.context.strokeStyle = "#DDF6F5";
        }

        this.drawOutboundBars(
          this.context,
          d[this.config.keyOut] as number,
          outboundSum,
          outboundSumPosition,
          yOffset,
          i
        );
        this.drawOutboundConnectingLines(
          this.context,
          d[this.config.keyOut] as number,
          outboundSum,
          outboundSumPosition,
          yOffset,
          i
        );

        // Draw the interaction context
        const interactionColor = this.genColor();
        this.figures.set(interactionColor, { index: i, outbound: true });
        this.interactionContext.fillStyle = interactionColor;
        this.interactionContext.strokeStyle = interactionColor;
        this.drawOutboundBars(
          this.interactionContext,
          d[this.config.keyOut] as number,
          outboundSum,
          outboundSumPosition,
          yOffset,
          i
        );
        this.drawOutboundConnectingLines(
          this.interactionContext,
          d[this.config.keyOut] as number,
          outboundSum,
          outboundSumPosition,
          yOffset,
          i
        );

        this.drawOutboundValueLabels(d, outboundSum, outboundSumPosition, yOffset, i);

        outboundSum += d[this.config.keyOut] as number;
      });

    this.data
      .filter((d) => (d[this.config.keyIn] as number) !== 0)
      .sort((a, b) => {
        return (b[this.config.keyIn] as number) - (a[this.config.keyIn] as number);
      })
      .forEach((d, i) => {
        this.context.fillStyle = this.config.inboundFill;
        this.context.strokeStyle = this.config.inboundStroke;
        if (this.mouseOver?.index === i && this.mouseOver?.outbound === false) {
          this.context.fillStyle = "#AAD6FF";
          this.context.strokeStyle = "#E7F3FF";
        }
        this.drawInboundBars(this.context, d[this.config.keyIn] as number, inboundSum, inboundSumPosition, yOffset, i);

        this.drawInboundConnectingLines(
          this.context,
          d[this.config.keyIn] as number,
          inboundSum,
          inboundSumPosition,
          yOffset,
          i
        );

        //Draw the interaction context
        const interactionColor = this.genColor();
        this.figures.set(interactionColor, { index: i, outbound: false });
        this.interactionContext.fillStyle = interactionColor;
        this.interactionContext.strokeStyle = interactionColor;
        this.drawInboundBars(
          this.interactionContext,
          d[this.config.keyIn] as number,
          inboundSum,
          inboundSumPosition,
          yOffset,
          i
        );
        this.drawInboundConnectingLines(
          this.interactionContext,
          d[this.config.keyIn] as number,
          inboundSum,
          inboundSumPosition,
          yOffset,
          i
        );

        this.drawInboundValueLabels(d, inboundSum, inboundSumPosition, yOffset, i);

        inboundSum += d[this.config.keyIn] as number;
      });

    // Draw the total outbound value label
    this.labelsContainer
      .append("div")
      .attr("class", `flow-outbound-value-total ${hoverOutboundClass}`)
      .attr("style", `top: 0px; left: ${outboundSumPosition - 5}px;`)
      .text(d3.format(",")(outboundSum));

    // Draw the total inbound value label
    this.labelsContainer
      .append("div")
      .attr("class", `flow-inbound-value-total ${hoverInboundClass}`)
      .attr("style", `top: 0px; left: ${inboundSumPosition + this.config.barWidth + 5}px;`)
      .text(d3.format(",")(inboundSum));

    return this;
  }
}

export default FlowChartCanvas;
