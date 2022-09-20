import { ReactElement, useLayoutEffect, useRef, useState } from "react";
export default function Collapse(props: { children: ReactElement; collapsed: boolean; animate: boolean }) {
  const ref = useRef<HTMLDivElement>(null);
  const [styleState, setStyleState] = useState({
    height: "auto",
    overflow: "hidden",
    transition: props.animate ? "height 0.3s" : "none",
  });
  useLayoutEffect(() => {
    if (ref.current) {
      setStyleState({ ...styleState, height: props.collapsed ? "0" : ref.current.scrollHeight + "px" });
    }
  }, [props]);
  return (
    <div ref={ref} style={styleState}>
      {props.children}
    </div>
  );
}
