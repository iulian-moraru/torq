import styles from "./progressHeader.module.scss";
import React, { MouseEventHandler, ReactEventHandler } from "react";
import classNames from "classnames";
import {
  Checkmark16Regular as CompletedIcon,
  Edit16Regular as ActiveIcon,
  Subtract16Regular as InactiveIcon,
  ArrowSync16Filled as ProcessingIcon,
  Dismiss16Regular as ErrorIcon,
} from "@fluentui/react-icons";

const progressStepClass = {
  0: styles.active,
  1: styles.completed,
  2: styles.disabled,
  3: styles.processing,
  4: styles.error,
};

const progressStepIcon = {
  0: <ActiveIcon />,
  1: <CompletedIcon />,
  2: <InactiveIcon />,
  3: <ProcessingIcon />,
  4: <ErrorIcon />,
};

export enum ProgressStepState {
  active = 0,
  completed = 1,
  disabled = 2,
  processing = 3,
  error = 4,
}

export type ProgressStep = {
  label: string;
  state: ProgressStepState;
  onClick?: MouseEventHandler<HTMLDivElement> | undefined;
  last?: boolean;
};

export function Step(props: ProgressStep) {
  return (
    <div
      className={classNames(styles.step, progressStepClass[props.state], { [styles.lastStep]: props.last })}
      onClick={props.onClick}
    >
      <div className={styles.stepLabel}>{props.label}</div>
      <div className={classNames(styles.stepIndicatorWrapper)}>
        <div className={styles.stepIcon}>{progressStepIcon[props.state]}</div>
        <div className={classNames(styles.stepLine)} />
      </div>
    </div>
  );
}

type progressHeaderProps = {
  modalCloseHandler: (e: ReactEventHandler) => void;
  children: Array<React.ReactNode> | React.ReactNode;
};

const cssGridTemplate = (stepsLength: number) => {
  return { gridTemplateColumns: `repeat(${stepsLength - 1}, 1fr) min-content` };
};

function ProgressHeader(props: progressHeaderProps) {
  return (
    <div className={styles.progressHeaderWrapper}>
      <div
        className={styles.progressHeaderContainer}
        style={cssGridTemplate(!Array.isArray(props.children) ? 1 : props.children.length)}
      >
        {!Array.isArray(props.children)
          ? props.children
          : props.children.map((step, index) => {
              return step;
            })}
      </div>
    </div>
  );
}

export default ProgressHeader;
