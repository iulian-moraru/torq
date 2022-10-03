import { MoneyHand24Regular as TransactionIconModal } from "@fluentui/react-icons";
import { useState } from "react";
import PopoutPageTemplate from "features/templates/popoutPageTemplate/PopoutPageTemplate";
import ChannelPage from "features/channel/ChannelPage";

type PopoutChannelsPageProps = {
  show: boolean;
  modalCloseHandler: () => void;
};

function PopoutChannelsPage(props: PopoutChannelsPageProps) {
  const [expandAdvancedOptions, setExpandAdvancedOptions] = useState(false);

  // const handleAdvancedToggle = () => {
  //   setExpandAdvancedOptions(!expandAdvancedOptions);
  // };

  return (
    <PopoutPageTemplate
      title={"Channel"}
      show={props.show}
      onClose={props.modalCloseHandler}
      icon={<TransactionIconModal />}
    >
      <ChannelPage />
    </PopoutPageTemplate>
  );
}

export default PopoutChannelsPage;
