# Week-of-year with Sunday as the first day of the week.
#
# weekOfYear lib dateStr
#   dateStr — "YYYYMMDD" string, or null to use the current date (approximate)
#   returns  — integer week number (1–54)
#
# Week 1 is the week (Sun–Sat) that contains January 1st.
# The anchor is the Sunday on or before Jan 1, which may fall in December
# of the prior year — exactly mirroring how US calendars are laid out.

lib:

let
  # Sakamoto's algorithm — fast day-of-week without epoch arithmetic.
  # Returns 0 = Sunday, 1 = Monday … 6 = Saturday.
  dayOfWeek = year: month: day:
    let
      offsets = [ 0 3 2 5 0 3 5 1 4 6 2 4 ];
      y       = if month < 3 then year - 1 else year;
    in
    lib.mod
      (y + builtins.div y 4 - builtins.div y 100 + builtins.div y 400
         + builtins.elemAt offsets (month - 1) + day)
      7;

  isLeap = y: (lib.mod y 4 == 0 && lib.mod y 100 != 0) || lib.mod y 400 == 0;

  # Cumulative days before the start of each month in a non-leap year.
  daysBeforeMonth = [ 0 31 59 90 120 151 181 212 243 273 304 334 ];

  fromDateStr = ds:
    let
      year  = lib.toInt (builtins.substring 0 4 ds);
      month = lib.toInt (lib.removePrefix "0" (builtins.substring 4 2 ds));
      day   = lib.toInt (lib.removePrefix "0" (builtins.substring 6 2 ds));

      leapOffset = if isLeap year && month > 2 then 1 else 0;
      dayOfYear  = builtins.elemAt daysBeforeMonth (month - 1) + leapOffset + day;

      # The Sunday on or before Jan 1 anchors week 1.  Its day-of-year can be
      # zero or negative when Jan 1 itself is not a Sunday.
      jan1DoW     = dayOfWeek year 1 1;  # 0 = Sunday, so no conversion needed
      week1Sunday = 1 - jan1DoW;
    in
    builtins.div (dayOfYear - week1Sunday) 7 + 1;

  # Approximate current week when no date string is provided.
  fromCurrentTime =
    let
      totalDays  = builtins.div builtins.currentTime 86400;
      daysInYear = y: if isLeap y then 366 else 365;
      yearResult = lib.foldl'
        (acc: y:
          if acc.done || acc.remaining < daysInYear y
          then acc // { done = true; }
          else acc // { remaining = acc.remaining - daysInYear y; })
        { remaining = totalDays; done = false; }
        (lib.range 1970 2200);
      dayOfYear = yearResult.remaining + 1;
    in
    builtins.div (dayOfYear - 1) 7 + 1;

in
dateStr: if dateStr == null then fromCurrentTime else fromDateStr dateStr
