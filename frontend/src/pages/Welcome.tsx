import { useEffect } from "react";
import { AccountSetupScreen } from "@/components/domain/AccountSetupScreen";

export function Welcome() {
  useEffect(() => {
    document.title = "Welcome — EnviToo";
  }, []);

  return <AccountSetupScreen />;
}