import classNames from "classnames";
import styles from "./button.module.scss";

export enum buttonColor {
  primary,
  secondary,
  ghost,
  warning,
  green,
  subtle,
}

export enum buttonPosition {
  center,
  fullWidth,
}

export enum buttonSize {
  medium,
  small,
  large,
}

const buttonPositionClass = {
  0: styles.positionLeft,
  1: styles.positionRight,
  2: styles.positionCenter,
  3: styles.positionFullWidth,
};

const buttonColorClass = {
  0: styles.primary,
  1: styles.secondary,
  2: styles.ghost,
  3: styles.warning,
  4: styles.green,
  5: styles.subtle,
};

const buttonSizeClass = {
  0: styles.medium,
  1: styles.small,
  2: styles.large,
};

export default function Button(props: {
  text?: string;
  type?: string;
  icon?: any;
  onClick?: () => void;
  className?: string;
  isOpen?: boolean;
  buttonColor: buttonColor;
  buttonPosition?: buttonPosition;
  buttonSize?: buttonSize;
  submit?: boolean;
  disabled?: boolean;
}) {
  const handleClick = () => {
    if (props.onClick) {
      props.onClick();
    }
  };
  return (
    <button
      type={props.submit ? "submit" : "button"}
      className={classNames(
        styles.button,
        props.className,
        buttonColorClass[props.buttonColor],
        buttonPositionClass[props.buttonPosition || 0],
        buttonSizeClass[props.buttonSize || 0],
        {
          [styles.open]: props.isOpen,
        }
      )}
      onClick={handleClick}
      disabled={props.disabled}
    >
      {props.icon && props.icon}
      {props.text && <div className="text">{props.text}</div>}
    </button>
  );
}

export function ButtonWrapper(props: {
  leftChildren?: Array<React.ReactNode> | React.ReactNode;
  rightChildren?: Array<React.ReactNode> | React.ReactNode;
  className?: string;
}) {
  return (
    <div className={classNames(styles.buttonWrapper, props.className)}>
      <div className={styles.leftButtonContainer}>{props.leftChildren}</div>
      <div className={styles.rightButtonContainer}>{props.rightChildren}</div>
    </div>
  );
}
