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
  left,
  right,
  center,
  fullWidth,
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

function Button(props: {
  text?: string;
  type?: string;
  icon?: any;
  onClick?: () => void;
  className?: string;
  isOpen?: boolean;
  buttonColor: buttonColor;
  buttonPosition?: buttonPosition;
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

export default Button;
