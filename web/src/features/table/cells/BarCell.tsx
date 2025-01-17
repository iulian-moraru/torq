import styles from "./cell.module.scss";
import { format } from "d3";
import React from "react";
import classNames from "classnames";

interface barCell {
  current: number;
  total: number;
  className?: string;
  showPercent?: boolean;
}

function formatPercent(num: number) {
  return new Intl.NumberFormat('default', {
    style: 'percent',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(num / 100);
}

const formatterDetailed = format(",.2f");
const formatter = format(",.0f");

function BarCell({ current, total, className, showPercent }: barCell) {
  const percent = (current || 0) / total;
  let data = current % 1 != 0 ? formatterDetailed(current) : formatter(current)

  if (showPercent) {
    data = formatPercent(current)
  }

  return (
    <div className={classNames(styles.cell, styles.barCell, className)}>
      <div className={styles.current}>{data}</div>
      <div className={styles.barWrapper}>
        <div className={styles.bar} style={{ width: percent * 100 + "%" }} />
        <div className={styles.totalBar} />
      </div>
    </div>
  );
}

const BarCellMemo = React.memo(BarCell);
export default BarCellMemo;
