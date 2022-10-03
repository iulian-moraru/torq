import React from "react";
import styles from "./channel-page.module.scss";
import * as d3 from "d3";
import classNames from "classnames";
import Popover from "../popover/Popover";
import TimeIntervalSelect from "../timeIntervalSelect/TimeIntervalSelect";
import ProfitsChart from "./revenueChart/ProfitsChart";
import EventsChart from "./eventsChart/EventsChart";
import EventsCard from "../eventsCard/EventsCard";
import Switch from "../inputs/Slider/Switch";
import Button, { buttonColor, buttonSize } from "../buttons/Button";
import Select from "../inputs/Select";
import { Flag16Regular as EventFlagIcon } from "@fluentui/react-icons";
import FlowChart from "./flowChart/FlowChart";
import { useGetFlowQuery, useGetChannelHistoryQuery } from "apiSlice";
import { useAppSelector, useAppDispatch } from "../../store/hooks";
import { selectTimeInterval } from "../timeIntervalSelect/timeIntervalSlice";
import { addDays, format } from "date-fns";
import { useParams } from "react-router";
import {
  selectEventChartKey,
  selectFlowKeys,
  selectProfitChartKey,
  updateEventChartKey,
  updateFlowKey,
  updateProfitChartKey,
  updateBalanceChanID,
  selectBalanceChanID,
} from "./channelSlice";
import BalanceChart from "./balanceChart/BalanceChart";
import DetailsPageTemplate from "features/templates/detailsPageTemplate/DetailsPageTemplate";
import { Link } from "react-router-dom";

const ft = d3.format(",.0f");

const eventNames = new Map([
  ["fee_rate", "Fee rate"],
  ["base_fee", "Base fee"],
  ["min_htlc", "Min htlc"],
  ["max_htlc", "Max htlc"],
  ["enabled", "Enabled"],
  ["disabled", "Disabled"],
]);

type ChannelPageProps = {
  chanId?: string;
};

function ChannelPage(props: ChannelPageProps) {
  const currentPeriod = useAppSelector(selectTimeInterval);
  const dispatch = useAppDispatch();
  const from = format(new Date(currentPeriod.from), "yyyy-MM-dd");
  const to = format(new Date(currentPeriod.to), "yyyy-MM-dd");
  const [allToggle, setAllToggle] = React.useState(true);
  const [selectedEvents, setSelectedEvents] = React.useState(
    new Map<string, boolean>([
      ["fee_rate", true],
      ["base_fee", true],
      ["min_htlc", true],
      ["max_htlc", true],
      ["enabled", true],
      ["disabled", true],
    ])
  );
  const handleSelectEventUpdate = (type: string) => {
    return (checked: boolean) => {
      setSelectedEvents(new Map(selectedEvents.set(type, checked)));
    };
  };

  const { chanId } = useParams();
  const { data, isLoading } = useGetFlowQuery({
    from: from,
    to: format(addDays(new Date(currentPeriod.to), 1), "yyyy-MM-dd"),
    chanId: chanId || "1",
  });
  const historyQuery = useGetChannelHistoryQuery({
    from: from,
    to: format(addDays(new Date(currentPeriod.to), 1), "yyyy-MM-dd"),
    chanIds: chanId || "1",
  });

  const flowKey = useAppSelector(selectFlowKeys);
  const profitKey = useAppSelector(selectProfitChartKey);
  const eventKey = useAppSelector(selectEventChartKey);
  let balanceChanId = useAppSelector(selectBalanceChanID);
  if (balanceChanId.label === "") {
    balanceChanId = { value: 0, label: historyQuery?.data?.channel_balance[0]?.LNDShortChannelId || "" };
  }

  let total_capacity = 0;
  if (historyQuery?.data?.channels) {
    total_capacity = historyQuery.data.channels
      .map((d: { capacity: number }) => {
        return d.capacity;
      })
      .reduce((partialSum: number, a: number) => partialSum + a, 0);
  }
  const profit: number =
    historyQuery?.data?.revenue_out - historyQuery?.data?.on_chain_cost - historyQuery?.data?.rebalancing_cost / 1000;

  const totalCost: number = historyQuery?.data?.on_chain_cost + historyQuery?.data?.rebalancing_cost / 1000;

  const selectedEventsCount = Array.from(selectedEvents).filter((d) => d[1]).length;
  const title =
    !isLoading &&
    historyQuery.data &&
    (historyQuery.data.channels || [])
      .map((d: any, _: number) => {
        return d.alias;
      })
      .filter((value: any, index: number, self: any[]) => {
        return self.indexOf(value) === index;
      })
      .join(", ");
  let channelBalanceOptions = [{ value: 0, label: "" }];
  if (historyQuery?.data?.channel_balance) {
    channelBalanceOptions = historyQuery.data.channel_balance.map((d: any, i: number) => {
      return { value: i, label: d.ChanId };
    });
  }
  const breadcrumbs = [
    <span key="b1">&quot;Analyse&quot;</span>,
    <span key="b2">&quot;Inspect&quot;</span>,
    <Link key="b3" to={`/analyse/inspect/${chanId}`}>
      {chanId}
    </Link>,
  ];
  return (
    <DetailsPageTemplate title={title} titleContent={<TimeIntervalSelect />} breadcrumbs={breadcrumbs}>
      <div className={styles.channelWrapper}>
        <div className={classNames(styles.pageRow, styles.channelSummary)}>
          <div className={styles.shortColumn}>
            <div className={styles.card}>
              <div className={styles.heading}>Revenue</div>
              <div className={styles.cardRow}>
                <div className={styles.rowLabel}>Forwarding fees</div>
                <div className={styles.rowValue}>{ft(historyQuery?.data?.revenue_out)}</div>
              </div>
              <div className={styles.cardRow}>
                <div className={styles.rowLabel}>Channel Leases</div>
                <div className={classNames(styles.rowValue, styles.comingSoon)}>(Coming soon)</div>
              </div>
              <div className={styles.cardRow}>
                <div className={styles.rowLabel}>Total</div>
                <div className={styles.rowValue}>{historyQuery?.data?.revenue_out}</div>
              </div>
            </div>
            <div className={styles.card}>
              <div className={styles.heading}>Expenses</div>
              <div className={styles.cardRow}>
                <div className={styles.rowLabel}>Rebalancing</div>
                <div className={classNames(styles.rowValue)}>{ft(historyQuery?.data?.rebalancing_cost / 1000)}</div>
              </div>
              <div className={styles.cardRow}>
                <div className={styles.rowLabel}>Open & Close</div>
                <div className={classNames(styles.rowValue)}>{ft(historyQuery?.data?.on_chain_cost)}</div>
              </div>
              <div className={styles.cardRow}>
                <div className={styles.rowLabel}>Total</div>
                <div className={classNames(styles.rowValue)}>
                  {ft(historyQuery?.data?.on_chain_cost + historyQuery?.data?.rebalancing_cost / 1000)}
                </div>
              </div>
            </div>
            <div className={styles.card}>
              <div className={styles.heading}>Profit</div>
              <div className={styles.cardRow}>
                <div className={styles.rowLabel}>Total</div>
                <div className={classNames(styles.rowValue)}>{ft(profit)}</div>
              </div>
              <div className={styles.cardRow}>
                <div className={styles.rowLabel}>Gross Profit Margin</div>
                <div className={classNames(styles.rowValue)}>
                  {d3.format(".2%")((historyQuery?.data?.revenue_out - totalCost) / historyQuery?.data?.revenue_out)}
                </div>
              </div>
              <div className={styles.cardRow}>
                <div className={styles.rowLabel}>Turnover</div>
                <div className={classNames(styles.rowValue)}>
                  {d3.format(",.2")(historyQuery?.data?.amount_total / total_capacity)}
                </div>
              </div>
            </div>
            <div className={styles.card}>
              <div className={styles.heading}>Automation</div>
              <div className={styles.sliderRow}>
                <div className={styles.rowLabel}>
                  <Switch label={"Fees"} />
                </div>
                <div className={classNames(styles.rowValue, styles.comingSoon)}>(Coming soon)</div>
              </div>
              <div className={styles.sliderRow}>
                <div className={styles.rowLabel}>
                  <Switch label={"Rebalancing"} />
                </div>
                <div className={classNames(styles.rowValue, styles.comingSoon)}>(Coming soon)</div>
              </div>
              <div className={styles.sliderRow}>
                <div className={styles.rowLabel}>
                  <Switch label={"HTLC amount"} />
                </div>
                <div className={classNames(styles.rowValue, styles.comingSoon)}>(Coming soon)</div>
              </div>
            </div>
          </div>

          <div className={classNames(styles.card, styles.channelSummaryChart)}>
            <div className={styles.profitChartControls}>
              <div className={styles.profitChartLeftControls}>
                <Select
                  className={"small"}
                  value={profitKey}
                  onChange={(newValue) => {
                    if (newValue) {
                      dispatch(
                        updateProfitChartKey({
                          key: (newValue as { value: string; label: string }) || {
                            value: "amount",
                            label: "Amount",
                          },
                        })
                      );
                    }
                  }}
                  options={[
                    { value: "amount", label: "Amount" },
                    { value: "revenue", label: "Revenue" },
                    { value: "count", label: "Count" },
                  ]}
                />
              </div>
              {/*<div className={styles.profitChartRightControls}>*/}
              {/*  <SettingsIcon />*/}
              {/*  Settings*/}
              {/*</div>*/}
            </div>
            <div className={styles.chartContainer}>
              {historyQuery.data && <ProfitsChart data={historyQuery.data.history} from={from} to={to} />}
            </div>
          </div>
        </div>

        <div className={classNames(styles.pageRow, styles.tripleRow)}>
          <div className={styles.card}>
            <div className={styles.heading}>Amount</div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Capacity</div>
              <div className={styles.rowValue}>{ft(total_capacity)}</div>
            </div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Outbound</div>
              <div className={styles.rowValue}>{ft(historyQuery?.data?.amount_out)}</div>
            </div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Inbound</div>
              <div className={classNames(styles.rowValue)}>{ft(historyQuery?.data?.amount_in)}</div>
            </div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Total</div>
              <div className={styles.rowValue}>{ft(historyQuery?.data?.amount_total)}</div>
            </div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Turnover</div>
              <div className={classNames(styles.rowValue)}>
                {d3.format(",.2")(historyQuery?.data?.amount_total / total_capacity)}
              </div>
            </div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Balance score</div>
              <div className={classNames(styles.rowValue)}>
                {d3.format(".1%")(
                  Math.min(historyQuery?.data?.amount_in, historyQuery?.data?.amount_out) /
                    Math.max(historyQuery?.data?.amount_in, historyQuery?.data?.amount_out)
                )}
              </div>
            </div>
          </div>

          <div className={styles.card}>
            <div className={styles.heading}>Revenue</div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Outbound</div>
              <div className={styles.rowValue}>{ft(historyQuery?.data?.revenue_out)}</div>
            </div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Inbound</div>
              <div className={classNames(styles.rowValue)}>{ft(historyQuery?.data?.revenue_in)}</div>
            </div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Total</div>
              <div className={styles.rowValue}>{ft(historyQuery?.data?.revenue_total)}</div>
            </div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Average fee out</div>
              <div className={classNames(styles.rowValue)}>
                {d3.format(",.1f")((historyQuery?.data?.revenue_out / historyQuery?.data?.amount_out) * 1000 * 1000)}
              </div>
            </div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Average fee in</div>
              <div className={classNames(styles.rowValue)}>
                {d3.format(",.1f")((historyQuery?.data?.revenue_in / historyQuery?.data?.amount_in) * 1000 * 1000)}
              </div>
            </div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Average fee</div>
              <div className={classNames(styles.rowValue)}>
                {d3.format(",.1f")(
                  (historyQuery?.data?.revenue_total / historyQuery?.data?.amount_total) * 1000 * 1000
                )}
              </div>
            </div>
          </div>

          <div className={styles.card}>
            <div className={styles.heading}>Transaction count</div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Outbound</div>
              <div className={styles.rowValue}>{ft(historyQuery?.data?.count_out)}</div>
            </div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Inbound</div>
              <div className={classNames(styles.rowValue)}>{ft(historyQuery?.data?.count_in)}</div>
            </div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Total</div>
              <div className={styles.rowValue}>{ft(historyQuery?.data?.count_total)}</div>
            </div>
            <div className={styles.cardRow}>
              <div className={styles.rowLabel}>Balance score (Nb. Tx)</div>
              <div className={classNames(styles.rowValue)}>
                {d3.format(".1%")(
                  Math.min(historyQuery?.data?.count_in, historyQuery?.data?.count_out) /
                    Math.max(historyQuery?.data?.count_in, historyQuery?.data?.count_out)
                )}
              </div>
            </div>
          </div>
        </div>

        <div className={classNames(styles.pageRow, styles.eventSummary)}>
          <div className={styles.shortColumn}>
            <EventsCard events={historyQuery} selectedEvents={selectedEvents} />
          </div>
          <div className={classNames(styles.card, styles.channelSummaryChart)} style={{ height: "600px" }}>
            <div className={styles.profitChartControls}>
              <div className={styles.profitChartLeftControls}>
                <Select
                  value={eventKey}
                  onChange={(newValue) => {
                    if (newValue) {
                      dispatch(
                        updateEventChartKey({
                          key: (newValue as { value: string; label: string }) || {
                            value: "amount",
                            label: "Amount",
                          },
                        })
                      );
                    }
                  }}
                  options={[
                    { value: "amount", label: "Amount" },
                    { value: "revenue", label: "Revenue" },
                    { value: "count", label: "Count" },
                  ]}
                />
              </div>
              <div className={styles.profitChartRightControls}>
                <Popover
                  button={
                    <Button
                      buttonColor={buttonColor.ghost}
                      buttonSize={buttonSize.small}
                      isOpen={selectedEventsCount > 0}
                      text={`${selectedEventsCount}`}
                      icon={<EventFlagIcon />}
                      className={"collapse-tablet"}
                    />
                  }
                  className={"right"}
                >
                  <div className={styles.channelChartSettingsPopover}>
                    <div className={styles.cardRow}>
                      <div className={styles.rowLabel}>
                        <Switch
                          label="Toggle all"
                          checked={allToggle}
                          onChange={(checked) => {
                            setAllToggle(checked);
                            setSelectedEvents(
                              new Map([
                                ["fee_rate", checked],
                                ["base_fee", checked],
                                ["min_htlc", checked],
                                ["max_htlc", checked],
                                ["enabled", checked],
                                ["disabled", checked],
                              ])
                            );
                          }}
                        />
                      </div>
                    </div>

                    {Array.from(selectedEvents).map((item) => {
                      return (
                        <div className={styles.cardRow} key={item[0]}>
                          <div className={styles.rowLabel}>
                            <Switch
                              label={eventNames.get(item[0]) || ""}
                              checked={selectedEvents.get(item[0])}
                              onChange={handleSelectEventUpdate(item[0])}
                            />
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </Popover>
              </div>
            </div>

            <div className={styles.chartContainer}>
              {historyQuery.data && (
                <EventsChart
                  from={from}
                  to={to}
                  data={historyQuery.data.history}
                  events={historyQuery.data.events}
                  selectedEventTypes={selectedEvents}
                />
              )}
            </div>
          </div>
        </div>

        <div className={classNames(styles.pageRow, styles.balanceChartWrapper)}>
          <div className={styles.card}>
            <div className={styles.profitChartControls}>
              <div className={styles.profitChartLeftControls}>
                <Select
                  value={balanceChanId}
                  onChange={(newValue) => {
                    if (newValue) {
                      dispatch(
                        updateBalanceChanID({
                          key: (newValue as { value: string; label: string }) || {
                            value: 0,
                            label: historyQuery.data.channel_balance[0].LNDShortChannelId,
                          },
                        })
                      );
                    }
                  }}
                  isDisabled={channelBalanceOptions.length < 2}
                  options={channelBalanceOptions}
                />
              </div>
            </div>
            <div className={classNames(styles.chartContainer)}>
              {!isLoading && historyQuery?.data?.channel_balance?.length && (
                <BalanceChart
                  data={historyQuery.data.channel_balance[balanceChanId.value].Balances}
                  totalCapacity={historyQuery.data.channels[balanceChanId.value].capacity}
                  from={from}
                  to={to}
                />
              )}
            </div>
          </div>
        </div>

        <div className={styles.pageRow}>
          <div className={styles.card}>
            <div className={styles.profitChartControls}>
              <div className={styles.profitChartLeftControls}>
                <Select
                  value={flowKey}
                  onChange={(newValue) => {
                    if (newValue) {
                      dispatch(
                        updateFlowKey({
                          flowKey: (newValue as { value: string; label: string }) || {
                            value: "amount",
                            label: "Amount",
                          },
                        })
                      );
                    }
                  }}
                  options={[
                    { value: "amount", label: "Amount" },
                    { value: "revenue", label: "Revenue" },
                    { value: "count", label: "Count" },
                  ]}
                />
              </div>
              <div className={styles.profitChartRightControls}>
                {/*<Popover*/}
                {/*  button={<Button text={"Settings"} icon={<SettingsIcon />} className={"collapse-tablet"} />}*/}
                {/*  className={"right"}*/}
                {/*>*/}
                {/*  Hello*/}
                {/*</Popover>*/}
              </div>
            </div>
            <div className="legendsContainer">
              <div className="sources">Sources</div>
              <div className="outbound">Outbound</div>
              <div className="inbound">Inbound</div>
              <div className="destinations">Destinations</div>
            </div>
            <div className={classNames(styles.chartWrapper, styles.flowChartWrapper)}>
              {!isLoading && data && <FlowChart data={data} />}
            </div>
          </div>
        </div>
      </div>
    </DetailsPageTemplate>
  );
}

export default ChannelPage;
