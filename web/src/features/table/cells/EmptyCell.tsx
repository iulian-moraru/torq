import styles from "./cell.module.scss";
import classNames from "classnames";

function EmptyCell(index?: number | string, className?: string) {
  return <div className={classNames(styles.cell, styles.empty, className)} key={"last-cell-" + index} />;
}

export default EmptyCell;
