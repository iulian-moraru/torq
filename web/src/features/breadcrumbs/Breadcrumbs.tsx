import React from "react";
import styles from "./breadcrumbs.module.scss";

type BreadcrumbsProps = {
  breadcrumbs: Array<string | React.ReactChildren>;
};

function Breadcrumbs(props: BreadcrumbsProps) {
  const totalLength = props.breadcrumbs.length - 1;
  return (
    <div className={styles.breadcrumbs}>
      {props.breadcrumbs.map((breadcrumb, i) => {
        const isLast = i === totalLength;
        return (
          <span key={i + "-breadcrumb"}>
            <>
              {breadcrumb}
              {!isLast && <span className={styles.breadcrumbSeparator}> / </span>}
            </>
          </span>
        );
      })}
    </div>
  );
}

export default Breadcrumbs;
