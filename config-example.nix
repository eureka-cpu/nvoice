{ pkgs ? import <nixpkgs> { } }:
{
  provider = {
    name = "Jane Smith";
    address = "123 Main St, Apt 4B\nNew York, NY 10001";
    email = "jane@example.com";
  };

  banks = [
    {
      name = "Example Bank ACH";
      note = "Preferred for domestic transfers";
      sections = [
        { label = "Account Holder"; value = "Jane Smith"; }
        { label = "Bank name and address"; value = "Example Bank\n456 Finance Ave\nNew York, NY 10002"; }
        { label = "Account Number"; value = "123456789"; position = "right"; }
        { label = "Routing Number"; value = "021000021"; position = "right"; }
      ];
    }
    {
      name = "Example Bank SWIFT";
      note = "For international transfers";
      sections = [
        { label = "Account Holder"; value = "Jane Smith"; }
        { label = "Bank name and address"; value = "Example Bank\n456 Finance Ave\nNew York, NY 10002"; }
        { label = "SWIFT/BIC"; value = "EXMPUS33XXX"; position = "right"; }
        { label = "IBAN"; value = "US12 3456 7890 1234 5678 90"; position = "right"; }
      ];
    }
  ];

  clients = {
    "Widgetry Inc." = {
      billing = "weekly";
    };
  };

  agencies = {
    "Acme Corp" = {
      emblem = ./emblem-example.svg;
      payment_terms = "Net 30";
    };
    "Example Agency LLC" = {
      payment_terms = "Net 30";
    };
  };
}
