import {
  ArrowSyncFilled as ProcessingIcon,
  CheckmarkRegular as SuccessIcon,
  DismissRegular as FailedIcon,
  MoneyHand24Regular as TransactionIconModal,
  Options20Regular as OptionsIcon,
} from "@fluentui/react-icons";
import { useGetDecodedInvoiceQuery, WS_URL } from "apiSlice";
import classNames from "classnames";
import { format } from "d3";
import Button, { buttonColor, ButtonWrapper } from "features/buttons/Button";
import TextInput from "features/forms/TextInput";
import ProgressHeader, { ProgressStepState, Step } from "features/progressTabs/ProgressHeader";
import ProgressTabs, { ProgressTabContainer } from "features/progressTabs/ProgressTab";
import { SectionContainer } from "features/section/SectionContainer";
import PopoutPageTemplate from "features/templates/popoutPageTemplate/PopoutPageTemplate";
import { ChangeEvent, useState } from "react";
import NumberFormat, { NumberFormatValues } from "react-number-format";
import { useNavigate } from "react-router";
import useWebSocket from "react-use-websocket";
import { NewPaymentError, NewPaymentResponse } from "../paymentTypes";
import styles from "./newPayments.module.scss";
import { PaymentProcessingErrors } from "./paymentErrorMessages";

const fd = format(",.0f");

export type NewPaymentRequest = {
  invoice: string;
  timeOutSecs: number;
  dest: string;
  amtMSat: number;
  feeLimitMsat: number;
  allowSelfPayment: boolean;
};

enum PaymentType {
  Unknown,
  Keysend,
  P2PKH,
  P2SH,
  P2WKH,
  P2TR,
  LightningMainnet,
  LightningTestnet,
  LightningSimnet,
  LightningRegtest,
}

const PaymentTypeLabel = {
  [PaymentType.Unknown]: "Unknown ",
  [PaymentType.P2PKH]: "Legacy Bitcoin ", // Legacy address
  [PaymentType.P2SH]: "Pay-to-Script-Hash ", // P2SH address
  [PaymentType.P2WKH]: "Segwit ", // Segwit address
  [PaymentType.P2TR]: "Taproot Address", // Taproot address
  [PaymentType.LightningMainnet]: "Mainnet Invoice",
  [PaymentType.LightningTestnet]: "Testnet Invoice",
  [PaymentType.LightningSimnet]: "Simnet Invoice",
  [PaymentType.LightningRegtest]: "Regtest Invoice",
  [PaymentType.Keysend]: "Keysend",
};

const paymentStatusClass = {
  IN_FLIGHT: styles.inFlight,
  FAILED: styles.failed,
  SUCCEEDED: styles.success,
};

const paymentStatusIcon = {
  IN_FLIGHT: <ProcessingIcon />,
  FAILED: <FailedIcon />,
  SUCCEEDED: <SuccessIcon />,
};

// RegEx used to check what type of destination the user enters.
// You can test them out here: https://regex101.com/r/OiXAlz/1
const LnPayrequestMainnetRegEx = /lnbc[0-9][0-9a-zA-Z]*/gm;
const LnPayrequestTestnetRegEx = /lntb[0-9][0-9a-zA-Z]*/gm;
const LnPayrequestSignetRegEx = /lnsb[0-9][0-9a-zA-Z]*/gm;
const LnPayrequestRegtestRegEx = /lnbcrt[0-9][0-9a-zA-Z]*/gm;
// const P2PKHAddressRegEx = /^1[0-9a-zA-Z]*/gm; // Legacy Adresses
const P2SHAddressRegEx = /^[3/r][0-9a-zA-Z]*/gm; // Pay to Script Hash
const P2WKHAddressRegEx = /^bc1q[0-9a-zA-Z]*/gm; // Segwit address
const P2TRAddressRegEx = /^bc1p[0-9a-zA-Z]*/gm; // Taproot address

const P2WKHAddressSignetRegEx = /^sb1q[0-9a-zA-Z]*/gm; // Segwit address
const P2TRAddressSignetRegEx = /^sb1p[0-9a-zA-Z]*/gm; // Taproot address

const LightningNodePubkeyRegEx = /^[0-9a-fA-F]{66}$/gm; // Keysend / Lightning Node Pubkey

function NewPaymentModal() {
  const [responses, setResponses] = useState<Array<NewPaymentResponse>>([]);

  const [destination, setDestination] = useState("");
  const [destinationType, setDestinationType] = useState<PaymentType>(0);
  const [paymentProcessingError, setPaymentProcessingError] = useState("");

  const [destState, setDestState] = useState(ProgressStepState.active);
  const [confirmState, setConfirmState] = useState(ProgressStepState.disabled);
  const [processState, setProcessState] = useState(ProgressStepState.disabled);
  const [stepIndex, setStepIndex] = useState(0);

  const [paymentDescription, setPaymentDescription] = useState("");

  function onNewPaymentMessage(event: MessageEvent<string>) {
    const response = JSON.parse(event.data);
    if (response?.type == "Error") {
      setPaymentProcessingError(response.error);
      onNewPaymentError(response as NewPaymentError);
      return;
    }
    onNewPaymentResponse(response as NewPaymentResponse);
  }

  function onNewPaymentResponse(message: NewPaymentResponse) {
    setResponses((prev) => [...prev, message]);
    if (message.status == "SUCCEEDED") {
      setProcessState(ProgressStepState.completed);
    } else if (message.status == "FAILED") {
      setPaymentProcessingError(message.failureReason);
      setProcessState(ProgressStepState.error);
    }
  }

  function onNewPaymentError(message: NewPaymentError) {
    console.log(PaymentProcessingErrors.get(message.error), message.error);
    setProcessState(ProgressStepState.error);
    console.log("error", message);
  }

  // This can also be an async getter function. See notes below on Async Urls.
  const { sendMessage, sendJsonMessage, lastMessage, lastJsonMessage, readyState, getWebSocket } = useWebSocket(
    WS_URL,
    {
      //Will attempt to reconnect on all close events, such as server shutting down
      shouldReconnect: (closeEvent) => true,
      share: true,
      onMessage: onNewPaymentMessage,
    }
  );

  const [expandAdvancedOptions, setExpandAdvancedOptions] = useState(false);
  const handleAdvancedToggle = () => {
    setExpandAdvancedOptions(!expandAdvancedOptions);
  };

  // LN advanced options
  const [allowSelfPayment, setAllowSelfPayment] = useState(true);
  const [feeLimit, setFeeLimit] = useState<number | undefined>(undefined);
  const [timeOutSecs, setTimeOutSecs] = useState(3600);
  const [amtSat, setAmtSat] = useState<number | undefined>(undefined);

  // On Chain advanced options
  const [satPerVbyte, setSatPerVbyte] = useState<number | undefined>(undefined);
  const [onChainPaymentAmount, setOnChainPaymentAmount] = useState<number>();
  const isLnInvoice = [
    PaymentType.LightningMainnet,
    PaymentType.LightningRegtest,
    PaymentType.LightningSimnet,
    PaymentType.LightningTestnet,
  ].includes(destinationType);

  const isOnChain = [PaymentType.P2SH, PaymentType.P2TR, PaymentType.P2WKH, PaymentType.P2PKH].includes(
    destinationType
  );

  // TODO: Get the estimated fee as well
  const decodedInvRes = useGetDecodedInvoiceQuery(
    { invoice: destination },
    {
      skip: !isLnInvoice,
    }
  );

  const closeAndReset = () => {
    setStepIndex(0);
    setDestState(ProgressStepState.active);
    setConfirmState(ProgressStepState.disabled);
    setProcessState(ProgressStepState.disabled);
  };

  const setDestinationHandler = (e: ChangeEvent<HTMLTextAreaElement>) => {
    if (e.target.value !== destination) {
      setDestinationType(0);
      decodedInvRes.data = undefined;
      setPaymentDescription("");
    }

    setDestination(e.target.value);
    if (e.target.value === "") {
      setDestinationType(0);
      setDestState(ProgressStepState.active);
      return;
    } else if (e.target.value.match(LnPayrequestMainnetRegEx)) {
      setDestinationType(PaymentType.LightningMainnet);
    } else if (e.target.value.match(LnPayrequestTestnetRegEx)) {
      setDestinationType(PaymentType.LightningTestnet);
    } else if (e.target.value.match(LnPayrequestSignetRegEx)) {
      setDestinationType(PaymentType.LightningSimnet);
    } else if (e.target.value.match(LnPayrequestRegtestRegEx)) {
      setDestinationType(PaymentType.LightningRegtest);
      // } else if (e.target.value.match(P2PKHAddressRegEx)) {
      //   setDestinationType(PaymentType.P2PKH);
      // } else if (e.target.value.match(P2SHAddressRegEx)) {
      //   // Pay to Script Hash
      //   setDestinationType(PaymentType.P2SH);
      // } else if (e.target.value.match(P2WKHAddressRegEx) || e.target.value.match(P2WKHAddressSignetRegEx)) {
      //   // Segwit address
      //   setDestinationType(PaymentType.P2WKH);
      // } else if (e.target.value.match(P2TRAddressRegEx) || e.target.value.match(P2TRAddressSignetRegEx)) {
      //   // Taproot
      //   setDestinationType(PaymentType.P2TR);
      //   // } else if (e.target.value.match(LightningNodePubkeyRegEx)) {
      //   //   setDestinationType(PaymentType.Keysend);
    } else {
      setDestinationType(PaymentType.Unknown);
      return;
    }

    // Prevent accidentally adding additional characters to the destination field after
    // the user has entered a valid destination by unfocusing (bluring) the input field.
    // e.target.blur();
    //setStepIndex(1);
    // setConfirmState(ProgressStepState.active);
    setDestState(ProgressStepState.completed);
  };

  const dynamicConfirmedState = () => {
    if (decodedInvRes.isLoading || decodedInvRes.isFetching) {
      return ProgressStepState.disabled;
    } else if (decodedInvRes.isError) {
      return ProgressStepState.disabled;
    }
    return confirmState;
  };

  const dynamicDestinationState = () => {
    if (decodedInvRes.isLoading || decodedInvRes.isFetching) {
      return ProgressStepState.processing;
    } else if (decodedInvRes.isError) {
      return ProgressStepState.error;
    }
    return destState;
  };

  function lnAmountField(amount: number) {
    if (amount > 0) {
      return fd(decodedInvRes.data ? decodedInvRes.data.valueMsat / 1000 : 0) + " sat";
    }
    return (
      <NumberFormat
        className={styles.amountInput}
        value={amtSat}
        placeholder={"0 sat"}
        onValueChange={(values) => setAmtSat(parseInt(values.value))}
        thousandSeparator=","
        suffix={" sat"}
      />
    );
  }

  const lnStep = (
    <ProgressTabContainer>
      <div className={styles.amountWrapper}>
        {/*<div className={styles.label}>You are paying</div>*/}
        {destinationType && (
          <span className={styles.destinationType}>{PaymentTypeLabel[destinationType] + " Detected"}</span>
        )}
        <div className={styles.amount}>{lnAmountField(decodedInvRes.data?.valueMsat)}</div>
        <div className={styles.label}>To</div>
        <div className={styles.destinationPreview}>{decodedInvRes?.data?.nodeAlias}</div>
      </div>
      {/*<div className={styles.destinationWrapper}>*/}
      {/*  <div className={styles.labelWrapper}>*/}
      {/*    <label htmlFor={"destination"} className={styles.destinationLabel}>*/}
      {/*      Description (only seen by you)*/}
      {/*    </label>*/}
      {/*  </div>*/}
      {/*  <textarea*/}
      {/*    id={"lnDescription"}*/}
      {/*    name={"lnDescription"}*/}
      {/*    className={styles.destinationTextArea}*/}
      {/*    autoComplete="off"*/}
      {/*    value={paymentDescription}*/}
      {/*    onChange={(e) => {*/}
      {/*      setPaymentDescription(e.target.value);*/}
      {/*    }}*/}
      {/*    rows={3}*/}
      {/*  />*/}
      {/*</div>*/}
      <SectionContainer
        title={"Advanced Options"}
        icon={OptionsIcon}
        expanded={expandAdvancedOptions}
        handleToggle={handleAdvancedToggle}
      >
        {/*<Switch*/}
        {/*  label={"Allow self payment"}*/}
        {/*  // labelPosition={"left"}*/}
        {/*  checked={allowSelfPayment}*/}
        {/*  onChange={(checked) => {*/}
        {/*    console.log("something");*/}
        {/*    setAllowSelfPayment(checked);*/}
        {/*  }}*/}
        {/*/>*/}
        <TextInput
          label={"Fee limit"}
          inputType={"number"}
          value={feeLimit}
          onChange={(e) => {
            setFeeLimit(e as number);
          }}
        />
        <TextInput label={"Timeout"} value={timeOutSecs} onChange={(e) => setTimeOutSecs(e as number)} />
      </SectionContainer>

      <ButtonWrapper
        className={styles.customButtonWrapperStyles}
        leftChildren={
          <Button
            text={"Back"}
            onClick={() => {
              setStepIndex(0);
              setDestState(ProgressStepState.completed);
              setConfirmState(ProgressStepState.active);
            }}
            buttonColor={buttonColor.ghost}
          />
        }
        rightChildren={
          <Button
            text={"Confirm"}
            onClick={() => {
              setStepIndex(2);
              setConfirmState(ProgressStepState.completed);
              setProcessState(ProgressStepState.processing);
              sendJsonMessage({
                reqId: "randId",
                type: "newPayment",
                NewPaymentRequest: {
                  // If the destination is not a pubkey, use it as an invoice
                  invoice: destination.match(LightningNodePubkeyRegEx) ? undefined : destination,
                  // If the destination is a pubkey send it as a dest input
                  dest: destination.match(LightningNodePubkeyRegEx) ? destination : undefined,
                  amtMSat: amtSat ? amtSat * 1000 : undefined, // 1 sat = 1000 msat
                  timeOutSecs: timeOutSecs,
                  feeLimitMsat: feeLimit ? feeLimit * 1000 : undefined,
                  allowSelfPayment: allowSelfPayment,
                },
              });
            }}
            disabled={!!decodedInvRes?.error}
            buttonColor={buttonColor.green}
          />
        }
      />
    </ProgressTabContainer>
  );

  const btcStep = (
    <ProgressTabContainer>
      <div className={styles.amountWrapper}>
        {destinationType && (
          <span className={styles.destinationType}>{PaymentTypeLabel[destinationType] + " Detected"}</span>
        )}
        <div className={styles.amount}>
          <NumberFormat
            className={styles.amountInput}
            suffix={" sat"}
            thousandSeparator=","
            value={onChainPaymentAmount}
            placeholder={"0 sat"}
            onValueChange={(values: NumberFormatValues) => {
              setOnChainPaymentAmount(values.floatValue || 0);
            }}
          />
        </div>
        <div className={styles.label}>To</div>
        <div className={styles.destinationPreview}>{destination}</div>
      </div>
      <div className={styles.destinationWrapper}>
        <div className={styles.labelWrapper}>
          <label htmlFor={"destination"} className={styles.destinationLabel}>
            Description (only seen by you)
          </label>
        </div>
        <textarea
          id={"lnDescription"}
          name={"lnDescription"}
          className={styles.destinationTextArea}
          autoComplete="off"
          value={paymentDescription}
          onChange={(e) => {
            setPaymentDescription(e.target.value);
          }}
          rows={3}
        />
      </div>
      <SectionContainer
        title={"Advanced Options"}
        icon={OptionsIcon}
        expanded={expandAdvancedOptions}
        handleToggle={handleAdvancedToggle}
      >
        <TextInput
          label={"Sat per vByte"}
          value={satPerVbyte}
          onChange={(value) => {
            setSatPerVbyte(value as number);
          }}
        />
      </SectionContainer>

      <ButtonWrapper
        className={styles.customButtonWrapperStyles}
        leftChildren={
          <Button
            text={"Back"}
            onClick={() => {
              setStepIndex(0);
              setDestState(ProgressStepState.completed);
              setConfirmState(ProgressStepState.active);
            }}
            buttonColor={buttonColor.ghost}
          />
        }
        rightChildren={
          <Button
            text={"Confirm"}
            onClick={() => {
              setStepIndex(2);
              setConfirmState(ProgressStepState.completed);
              setProcessState(ProgressStepState.processing);
            }}
            buttonColor={buttonColor.green}
          />
        }
      />
    </ProgressTabContainer>
  );

  const navigate = useNavigate();

  function destiationLabel() {
    if (decodedInvRes?.isError == false) {
      return PaymentTypeLabel[destinationType] + " Detected";
    }
    return "Can't decode invoice";
  }

  return (
    <PopoutPageTemplate title={"New Payment"} show={true} onClose={() => navigate(-1)} icon={<TransactionIconModal />}>
      <ProgressHeader modalCloseHandler={closeAndReset}>
        <Step label={"Destination"} state={dynamicDestinationState()} last={false} />
        <Step label={"Details"} state={dynamicConfirmedState()} last={false} />
        <Step label={"Process"} state={processState} last={true} />
      </ProgressHeader>

      <ProgressTabs showTabIndex={stepIndex}>
        <ProgressTabContainer>
          <div className={styles.destination}>
            <div className={styles.destinationWrapper}>
              <div className={styles.labelWrapper}>
                <label htmlFor={"destination"} className={styles.destinationLabel}>
                  Destination
                </label>
                {destination && (
                  <span
                    className={classNames(styles.destinationType, {
                      [styles.unknownAddressType]: destinationType === PaymentType.Unknown,
                      [styles.errorDestination]: decodedInvRes.isError,
                    })}
                  >
                    {destiationLabel()}
                  </span>
                )}
              </div>
              <textarea
                id={"destination"}
                name={"destination"}
                placeholder={"E.g. Lightning Invoice"} // , PubKey or On-chain Address
                className={styles.destinationTextArea}
                value={destination}
                onChange={setDestinationHandler}
                rows={6}
              />
            </div>
          </div>
          {/*<Note title={"Note:"}>*/}
          {/*  <span>*/}
          {/*    Torq will detect the transaction type based on the destination you enter. Valid destinations are on-chain*/}
          {/*    addresses, lightning invoices, and lightning node public keys for keysend.*/}
          {/*  </span>*/}
          {/*</Note>*/}
          <ButtonWrapper
            className={styles.customButtonWrapperStyles}
            rightChildren={
              <Button
                text={"Next"}
                disabled={!destinationType || decodedInvRes.isError}
                onClick={() => {
                  if (destination) {
                    setStepIndex(1);
                    setDestState(ProgressStepState.completed);
                    setConfirmState(ProgressStepState.active);
                  }
                }}
                buttonColor={decodedInvRes?.isError ? buttonColor.warning : buttonColor.subtle}
              />
            }
          />
        </ProgressTabContainer>
        {isLnInvoice && lnStep}
        {isOnChain && btcStep}
        <ProgressTabContainer>
          <div
            className={classNames(
              styles.paymentResultIconWrapper,
              { [styles.failed]: responses.length === 0 },
              paymentStatusClass[responses[responses.length - 1]?.status as "SUCCEEDED" | "FAILED" | "IN_FLIGHT"]
            )}
          >
            {" "}
            {responses.length === 0 && paymentStatusIcon["FAILED"]}
            {paymentStatusIcon[responses[responses.length - 1]?.status as "SUCCEEDED" | "FAILED" | "IN_FLIGHT"]}
          </div>
          <div className={classNames(styles.paymentStatusMessage)}>
            {PaymentProcessingErrors.get(paymentProcessingError) || paymentProcessingError}
          </div>
          <ButtonWrapper
            className={styles.customButtonWrapperStyles}
            rightChildren={
              <Button
                text={"New Payment"}
                onClick={() => {
                  setDestinationType(0);
                  setDestination("");
                  setDestState(ProgressStepState.active);
                  setConfirmState(ProgressStepState.disabled);
                  setProcessState(ProgressStepState.disabled);
                  setTimeOutSecs(3600);
                  setAmtSat(undefined);
                  setFeeLimit(undefined);
                  setStepIndex(0);
                  setResponses([]);
                }}
                buttonColor={buttonColor.subtle}
              />
            }
          />
        </ProgressTabContainer>
      </ProgressTabs>
    </PopoutPageTemplate>
  );
}

export default NewPaymentModal;
