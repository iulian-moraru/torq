import "./popover.scss";
import React, { ReactChild, SetStateAction, useEffect, Dispatch, useRef, useState } from "react";
import classNames from "classnames";

function useOutsideClose(ref: any, setIsPopoverOpen: Dispatch<SetStateAction<boolean>>) {
  useEffect(() => {
    function handleClickOutside(event: any) {
      if (ref.current && !ref.current.contains(event.target)) {
        setIsPopoverOpen(false);
      }
    }
    // Bind the event listener
    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      // Unbind the event listener on clean up
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [ref]);
}

interface PopoverInterface {
  className?: string;
  button?: ReactChild;
  children?: ReactChild;
}

const PopoverButton = React.forwardRef(function popoverButton({ className, button, children }: PopoverInterface, ref) {
  const wrapperRef = useRef(null);
  const [isPopoverOpen, setIsPopoverOpen] = useState(false);

  useOutsideClose(wrapperRef, setIsPopoverOpen);

  React.useImperativeHandle(ref, () => ({
    close() {
      setIsPopoverOpen(false);
    },
  }));

  return (
    <div
      onClick={() => setIsPopoverOpen(!isPopoverOpen)}
      ref={wrapperRef}
      className={classNames("torq-popover-button-wrapper", className)}
    >
      {button ? button : "button"}
      <div
        className={classNames("popover-wrapper right", {
          "popover-open": isPopoverOpen,
        })}
        onClick={(e) => {
          e.stopPropagation();
        }}
      >
        <div className={"popover-mobile-dismiss"}>
          <div className="left-container" onClick={(e) => e.stopPropagation()}>
            {button ? button : ""}
          </div>
          <div className="right-container dismiss-button" onClick={() => setIsPopoverOpen(false)}>
            Close
          </div>
        </div>
        {isPopoverOpen && <div className="popover-container">{children}</div>}
      </div>
    </div>
  );
});
export default PopoverButton;
