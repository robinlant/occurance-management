package i18n

import "github.com/gin-gonic/gin"

// Translations maps lang -> key -> translated string.
type Translations map[string]map[string]string

// translations holds all built-in translations.
var translations = Translations{
	"en": en,
	"de": de,
}

// T returns the translation for the given language and key.
// It falls back to English, then to the key itself.
func T(lang, key string) string {
	if m, ok := translations[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	// Fallback to English
	if lang != "en" {
		if m, ok := translations["en"]; ok {
			if v, ok := m[key]; ok {
				return v
			}
		}
	}
	// Fallback to the key itself
	return key
}

// GetLang reads the language preference from the "dr-lang" cookie.
// Defaults to "de" (German) since this is a German company app.
func GetLang(c *gin.Context) string {
	lang, err := c.Cookie("dr-lang")
	if err != nil || (lang != "en" && lang != "de") {
		return "de"
	}
	return lang
}

// --- English translations ---

var en = map[string]string{
	// Navigation
	"nav.dashboard":    "Dashboard",
	"nav.occurrences":  "Occurrences",
	"nav.calendar":     "Calendar",
	"nav.leaderboard":  "Leaderboard",
	"nav.groups":       "Groups",
	"nav.users":        "Users",
	"nav.settings":     "Settings",
	"nav.logout":       "Logout",
	"nav.search":       "Search people & occurrences...",
	"nav.toggleTheme":  "Toggle theme",
	"nav.skipContent":  "Skip to content",
	"nav.toggleNav":    "Toggle navigation",

	// Network error
	"error.network": "Network error. Please check your connection and try again.",

	// Occurrences page
	"occ.all":           "All",
	"occ.new":           "+ New occurrence",
	"occ.none":          "No occurrences",
	"occ.noneInGroup":   "in this group",
	"occ.past":          "past",
	"occ.needsPeople":   "needs people",
	"occ.filled":        "filled",
	"occ.overStaffed":   "over-staffed",
	"occ.people":        "people",
	"occ.overMax":       "over max",

	// Occurrence detail
	"occ.edit":              "Edit",
	"occ.delete":            "Delete",
	"occ.confirmDelete":     "Are you sure you want to delete this occurrence?",
	"occ.participants":      "Participants",
	"occ.noParticipants":    "No participants yet.",
	"occ.assignParticipant": "Assign participant",
	"occ.loading":           "Loading\u2026",
	"occ.signup":            "Sign up",
	"occ.withdraw":          "Withdraw",
	"occ.remove":            "remove",
	"occ.noAvailableUsers":  "No available users for this date.",
	"occ.searchUser":        "Search user...",
	"occ.assign":            "Assign",
	"occ.done":              "done",

	// Occurrence form
	"form.title":           "Title",
	"form.description":     "Description",
	"form.descPlaceholder": "Optional details\u2026",
	"form.group":           "Group (optional)",
	"form.selectGroup":     "Select group...",
	"form.none":            "\u2014 none \u2014",
	"form.dateTime":        "Date & Time",
	"form.minParticipants": "Min participants",
	"form.maxParticipants": "Max participants",
	"form.saveChanges":     "Save changes",
	"form.create":          "Create",
	"form.cancel":          "Cancel",

	// Calendar
	"cal.allEvents":   "All events",
	"cal.needsPeople": "Needs people",
	"cal.filled":      "Filled",
	"cal.overStaffed": "Over-staffed",
	"cal.allGroups":   "All groups",
	"cal.today":       "Today",
	"cal.selectDay":   "Select a day",
	"cal.clickDay":    "Click any day to see occurrences.",
	"cal.more":        "more",
	"cal.noOcc":       "No occurrences.",
	"cal.mon":         "Mon",
	"cal.tue":         "Tue",
	"cal.wed":         "Wed",
	"cal.thu":         "Thu",
	"cal.fri":         "Fri",
	"cal.sat":         "Sat",
	"cal.sun":         "Sun",

	// Dashboard
	"dash.openSpots":     "Open spots",
	"dash.allFilled":     "All filled up.",
	"dash.needsMore":     "needs",
	"dash.more":          "more",
	"dash.today":         "Today",
	"dash.tomorrow":      "Tomorrow",
	"dash.daysAgo":       "days ago",
	"dash.inDays":        "in",
	"dash.days":          "days",
	"dash.topPerformers": "Top performers",
	"dash.noData":        "No data yet.",

	// Profile
	"profile.totalParticipations": "Total participations",
	"profile.currentStreak":       "Current streak (months)",
	"profile.longestStreak":       "Longest streak (months)",
	"profile.activity":            "Activity \u2014 last 12 months",
	"profile.perMonth":            "per month",
	"profile.account":             "Account",
	"profile.changePassword":      "Change password",
	"profile.newPassword":         "New password",
	"profile.save":                "Save",
	"profile.outOfOffice":         "Out of office",
	"profile.noOOO":               "No out-of-office periods.",
	"profile.from":                "From",
	"profile.to":                  "To",
	"profile.addPeriod":           "Add period",
	"profile.remove":              "remove",
	"profile.less":                "Less",
	"profile.more":                "More",
	"profile.ooo":                 "OOO",
	"profile.currentlyOOO":        "Currently OOO",
	"profile.userInfo":            "User info",

	// Leaderboard
	"lb.from":             "From",
	"lb.to":               "To",
	"lb.filter":           "Filter",
	"lb.reset":            "Reset",
	"lb.thisYear":         "This year",
	"lb.studentYear":      "Student year",
	"lb.average":          "Average",
	"lb.noData":           "No data.",
	"lb.participants":     "Participants",
	"lb.organizers":       "Organizers",
	"lb.all":              "All",
	"lb.allGroups":        "All groups",

	// Groups
	"groups.createGroup":    "Create group",
	"groups.newGroupName":   "New group name",
	"groups.create":         "Create",
	"groups.delete":         "Delete",
	"groups.noGroups":       "No groups yet.",
	"groups.confirmDelete":  "Are you sure you want to delete the group",
	"groups.cannotBeUndone": "This cannot be undone.",

	// Users admin
	"users.allUsers":       "All users",
	"users.searchUsers":    "Search users...",
	"users.createUser":     "Create user",
	"users.name":           "Name",
	"users.fullName":       "Full name",
	"users.email":          "Email",
	"users.password":       "Password",
	"users.minPassword":    "Minimum 8 characters",
	"users.role":           "Role",
	"users.delete":         "Delete",
	"users.confirmDelete":  "Are you sure you want to delete this user?",
	"users.setNewPassword": "Set new password",
	"users.setNewEmail":    "Set new email",
	"users.viewProfile":    "Profile",
	"users.set":            "Set",
	"users.participant":    "Participant",
	"users.organizer":      "Organizer",
	"users.admin":          "Admin",

	// Settings
	"settings.emailConfig":          "Email Configuration",
	"settings.enableNotifications":  "Enable email notifications",
	"settings.smtpServer":           "SMTP Server",
	"settings.smtpHost":             "SMTP Host",
	"settings.port":                 "Port",
	"settings.smtpUsername":         "SMTP Username",
	"settings.smtpPassword":         "SMTP Password",
	"settings.sender":               "Sender",
	"settings.senderName":           "Sender Name",
	"settings.senderEmail":          "Sender Email",
	"settings.notificationLimits":   "Notification Limits",
	"settings.maxEmails":            "Max Emails Per Day (per user)",
	"settings.reminderDays":         "Upcoming Reminder Days",
	"settings.saveSettings":         "Save settings",
	"settings.testEmail":            "Test Email",
	"settings.testEmailDesc":        "Send a test email to your own address to verify the SMTP configuration works correctly.",
	"settings.sendTestEmail":        "Send test email",
	"settings.howItWorks":           "How It Works",
	"settings.howItWorksDesc":       "DutyRound sends email notifications automatically in the background.",
	"settings.newOccurrence":        "New Occurrence",
	"settings.newOccurrenceDesc":    "Participants receive a digest of available occurrences with open spots.",
	"settings.unfilledParticipants": "Unfilled Spots (Participants)",
	"settings.unfilledPartDesc":     "Participants are notified about upcoming occurrences that still need people.",
	"settings.unfilledOrganizers":   "Unfilled Spots (Organizers)",
	"settings.unfilledOrgDesc":      "Organizers receive a heads-up about upcoming occurrences with free places.",
	"settings.note":                 "Note:",
	"settings.noteDesc":             "Admins do not receive any notification emails. The \"Max Emails Per Day\" setting prevents spamming users.",

	// Login
	"login.email":    "Email",
	"login.password": "Password",
	"login.signIn":   "Sign in",

	// Error page
	"error.notFound":       "Not Found",
	"error.pageNotFound":   "Page not found",
	"error.goHome":         "Go home",
	"error.somethingWrong": "Something went wrong",

	// Flash messages (handlers)
	"flash.invalidEmailOrPassword":  "Invalid email or password.",
	"flash.invalidFormData":         "Invalid form data.",
	"flash.failedCreateOccurrence":  "Failed to create occurrence.",
	"flash.occurrenceCreated":       "Occurrence created.",
	"flash.failedUpdateOccurrence":  "Failed to update occurrence.",
	"flash.occurrenceUpdated":       "Occurrence updated.",
	"flash.failedDeleteOccurrence":  "Failed to delete occurrence.",
	"flash.occurrenceDeleted":       "Occurrence deleted.",
	"flash.passwordTooShort":        "Password must be at least 8 characters.",
	"flash.failedUpdatePassword":    "Failed to update password.",
	"flash.passwordUpdated":         "Password updated.",
	"flash.invalidDates":            "Invalid dates.",
	"flash.endDateAfterStart":       "End date must be after start date.",
	"flash.oooConflict":             "You have participations assigned in that period.",
	"flash.nameRequired":            "Name is required.",
	"flash.failedCreateGroup":       "Failed to create group.",
	"flash.groupCreated":            "Group created.",
	"flash.failedDeleteGroup":       "Failed to delete group.",
	"flash.groupDeleted":            "Group deleted.",
	"flash.failedCreateUser":        "Failed to create user.",
	"flash.userCreated":             "User created.",
	"flash.invalidRole":             "Invalid role selected.",
	"flash.failedSetPassword":       "Failed to set password.",
	"flash.failedDeleteUser":        "Failed to delete user.",
	"flash.userDeleted":             "User deleted.",
	"flash.cannotDeleteSelf":        "You cannot delete your own account.",
	"flash.failedSaveSettings":      "Failed to save settings.",
	"flash.settingsSaved":           "Email settings saved.",
	"flash.failedSendTestEmail":     "Failed to send test email: ",
	"flash.testEmailSent":           "Test email sent to ",
	"flash.userOOOSignup":           "You are out of office on this date.",
	"flash.userOOOAssign":           "User is out of office on this date.",
	"flash.dateCantBePast":          "Date cannot be in the past.",
	"flash.titleRequired":           "Title is required.",
	"flash.participantsMin":         "Participants must be at least 1.",
	"flash.minExceedsMax":           "Min participants cannot exceed max.",
	"flash.occurrenceFull":          "This occurrence is full.",
	"flash.emailAlreadyInUse":       "That email is already in use.",
	"flash.failedUpdateEmail":       "Failed to update email.",
	"flash.emailUpdated":            "Email updated.",
	"flash.commentTooLong":          "Comment must be 1000 characters or fewer.",
	"flash.oooConflictDetail":       "You are signed up for one or more occurrences during this period. Please withdraw from them first.",

	// Page titles
	"title.dashboard":      "Dashboard",
	"title.occurrences":    "Occurrences",
	"title.newOccurrence":  "New Occurrence",
	"title.editOccurrence": "Edit Occurrence",
	"title.calendar":       "Calendar",
	"title.leaderboard":    "Leaderboard",
	"title.groups":         "Groups",
	"title.users":          "Users",
	"title.settings":       "Email Settings",
	"title.profile":        "Profile",

	// Search results
	"search.noResults":   "No results for",
	"search.people":      "People",
	"search.occurrences": "Occurrences",

	// Occurrence over-limit
	"occ.full":                "Full",
	"occ.overLimitWarning":    "This occurrence is at capacity. Signing up will exceed the limit.",
	"occ.signupOverLimit":     "Sign up (over limit)",
	"form.allowOverLimit":     "Allow over-limit registrations",

	// Language
	"lang.switch": "DE",

	// Relative date (used in templates via relativeDay function)
	"rel.today":     "Today",
	"rel.yesterday": "Yesterday",
	"rel.tomorrow":  "Tomorrow",
	"rel.inDays":    "in %d days",
	"rel.daysAgo":   "%d days ago",

	// Dashboard extra
	"dash.yourSchedule":  "Your schedule",
	"dash.noUpcoming":    "No upcoming occurrences.",
	"dash.browseSpots":   "Browse open spots",

	// Occurrences list extra
	"occ.upcoming": "Upcoming",
	"occ.allGroups": "All groups",
	"occ.listView":  "List",
	"occ.cardView":  "Cards",

	// Occurrence form extra
	"form.copyFromExisting":      "Use as template",
	"form.copyFromExistingLabel": "Pre-fill title, description, time &amp; participant limits from another occurrence",
	"form.copySearchPlaceholder": "Search occurrences\u2026",

	// Comments
	"comment.title":         "Comments",
	"comment.none":          "No comments yet.",
	"comment.placeholder":   "Add a comment\u2026",
	"comment.post":          "Post",
	"comment.delete":        "delete",
	"comment.confirmDelete": "Delete this comment?",

	// Email content
	"email.hi":                     "Hi",
	"email.newOccAvailable":        "New occurrences are available. Sign up if you have time!",
	"email.headerOccurrence":       "Occurrence",
	"email.headerDate":             "Date",
	"email.headerSignedUp":         "Signed Up",
	"email.headerSpots":            "Spots",
	"email.headerPeopleNeeded":     "People Needed",
	"email.headerDaysUntil":        "Days Until",
	"email.headerStillNeeded":      "Still Needed",
	"email.headerCurrent":          "Current",
	"email.left":                   "left",
	"email.needed":                 "needed",
	"email.days":                   "days",
	"email.unfilledParticipantMsg": "The following upcoming occurrences still need people. Take one if you have time!",
	"email.unfilledOrganizerMsg":   "The following upcoming occurrences still have free places that need to be filled.",
	"email.testSubject":            "DutyRound - Test Email",
	"email.testTitle":              "Test Email",
	"email.testBody":               "If you are reading this, your DutyRound email configuration is working correctly.",
	"email.testVerified":           "SMTP configuration verified",
	"email.footer":                 "This is an automated notification from DutyRound. You can manage your notification preferences with your administrator.",
	"email.subjectNew":             "New occurrences available - DutyRound",
	"email.subjectUnfilledPart":    "Upcoming occurrences need people - DutyRound",
	"email.subjectUnfilledOrg":     "Upcoming occurrences still have free places - DutyRound",
}

// --- German translations ---

var de = map[string]string{
	// Navigation
	"nav.dashboard":    "Dashboard",
	"nav.occurrences":  "Eins\u00e4tze",
	"nav.calendar":     "Kalender",
	"nav.leaderboard":  "Rangliste",
	"nav.groups":       "Gruppen",
	"nav.users":        "Benutzer",
	"nav.settings":     "Einstellungen",
	"nav.logout":       "Abmelden",
	"nav.search":       "Personen & Eins\u00e4tze suchen...",
	"nav.toggleTheme":  "Design wechseln",
	"nav.skipContent":  "Zum Inhalt springen",
	"nav.toggleNav":    "Navigation umschalten",

	// Network error
	"error.network": "Netzwerkfehler. Bitte pr\u00fcfen Sie Ihre Verbindung.",

	// Occurrences page
	"occ.all":           "Alle",
	"occ.new":           "+ Neuer Einsatz",
	"occ.none":          "Keine Eins\u00e4tze",
	"occ.noneInGroup":   "in dieser Gruppe",
	"occ.past":          "vergangen",
	"occ.needsPeople":   "braucht Leute",
	"occ.filled":        "besetzt",
	"occ.overStaffed":   "\u00fcberbesetzt",
	"occ.people":        "Personen",
	"occ.overMax":       "\u00fcber Maximum",

	// Occurrence detail
	"occ.edit":              "Bearbeiten",
	"occ.delete":            "L\u00f6schen",
	"occ.confirmDelete":     "Sind Sie sicher, dass Sie diesen Einsatz l\u00f6schen m\u00f6chten?",
	"occ.participants":      "Teilnehmer",
	"occ.noParticipants":    "Noch keine Teilnehmer.",
	"occ.assignParticipant": "Teilnehmer zuweisen",
	"occ.loading":           "Laden\u2026",
	"occ.signup":            "Eintragen",
	"occ.withdraw":          "Austragen",
	"occ.remove":            "entfernen",
	"occ.noAvailableUsers":  "Keine verf\u00fcgbaren Benutzer f\u00fcr dieses Datum.",
	"occ.searchUser":        "Benutzer suchen...",
	"occ.assign":            "Zuweisen",
	"occ.done":              "erledigt",

	// Occurrence form
	"form.title":           "Titel",
	"form.description":     "Beschreibung",
	"form.descPlaceholder": "Optionale Details\u2026",
	"form.group":           "Gruppe (optional)",
	"form.selectGroup":     "Gruppe ausw\u00e4hlen...",
	"form.none":            "\u2014 keine \u2014",
	"form.dateTime":        "Datum & Uhrzeit",
	"form.minParticipants": "Min. Teilnehmer",
	"form.maxParticipants": "Max. Teilnehmer",
	"form.saveChanges":     "\u00c4nderungen speichern",
	"form.create":          "Erstellen",
	"form.cancel":          "Abbrechen",

	// Calendar
	"cal.allEvents":   "Alle Termine",
	"cal.needsPeople": "Braucht Leute",
	"cal.filled":      "Besetzt",
	"cal.overStaffed": "\u00dcberbesetzt",
	"cal.allGroups":   "Alle Gruppen",
	"cal.today":       "Heute",
	"cal.selectDay":   "Tag ausw\u00e4hlen",
	"cal.clickDay":    "Klicken Sie auf einen Tag, um Eins\u00e4tze zu sehen.",
	"cal.more":        "mehr",
	"cal.noOcc":       "Keine Eins\u00e4tze.",
	"cal.mon":         "Mo",
	"cal.tue":         "Di",
	"cal.wed":         "Mi",
	"cal.thu":         "Do",
	"cal.fri":         "Fr",
	"cal.sat":         "Sa",
	"cal.sun":         "So",

	// Dashboard
	"dash.openSpots":     "Offene Pl\u00e4tze",
	"dash.allFilled":     "Alle besetzt.",
	"dash.needsMore":     "braucht noch",
	"dash.more":          "",
	"dash.today":         "Heute",
	"dash.tomorrow":      "Morgen",
	"dash.daysAgo":       "Tagen",
	"dash.inDays":        "in",
	"dash.days":          "Tagen",
	"dash.topPerformers": "Top-Teilnehmer",
	"dash.noData":        "Noch keine Daten.",

	// Profile
	"profile.totalParticipations": "Gesamtteilnahmen",
	"profile.currentStreak":       "Aktuelle Serie (Monate)",
	"profile.longestStreak":       "L\u00e4ngste Serie (Monate)",
	"profile.activity":            "Aktivit\u00e4t \u2014 letzte 12 Monate",
	"profile.perMonth":            "pro Monat",
	"profile.account":             "Konto",
	"profile.changePassword":      "Passwort \u00e4ndern",
	"profile.newPassword":         "Neues Passwort",
	"profile.save":                "Speichern",
	"profile.outOfOffice":         "Abwesenheit",
	"profile.noOOO":               "Keine Abwesenheitszeitr\u00e4ume.",
	"profile.from":                "Von",
	"profile.to":                  "Bis",
	"profile.addPeriod":           "Zeitraum hinzuf\u00fcgen",
	"profile.remove":              "entfernen",
	"profile.less":                "Weniger",
	"profile.more":                "Mehr",
	"profile.ooo":                 "Abwesend",
	"profile.currentlyOOO":        "Derzeit abwesend",
	"profile.userInfo":            "Benutzerinfo",

	// Leaderboard
	"lb.from":             "Von",
	"lb.to":               "Bis",
	"lb.filter":           "Filtern",
	"lb.reset":            "Zur\u00fccksetzen",
	"lb.thisYear":         "Dieses Jahr",
	"lb.studentYear":      "Schuljahr",
	"lb.average":          "Durchschnitt",
	"lb.noData":           "Keine Daten.",
	"lb.participants":     "Teilnehmer",
	"lb.organizers":       "Organisatoren",
	"lb.all":              "Alle",
	"lb.allGroups":        "Alle Gruppen",

	// Groups
	"groups.createGroup":    "Gruppe erstellen",
	"groups.newGroupName":   "Neuer Gruppenname",
	"groups.create":         "Erstellen",
	"groups.delete":         "L\u00f6schen",
	"groups.noGroups":       "Noch keine Gruppen.",
	"groups.confirmDelete":  "Sind Sie sicher, dass Sie die Gruppe l\u00f6schen m\u00f6chten:",
	"groups.cannotBeUndone": "Dies kann nicht r\u00fcckg\u00e4ngig gemacht werden.",

	// Users admin
	"users.allUsers":       "Alle Benutzer",
	"users.searchUsers":    "Benutzer suchen...",
	"users.createUser":     "Benutzer erstellen",
	"users.name":           "Name",
	"users.fullName":       "Vollst\u00e4ndiger Name",
	"users.email":          "E-Mail",
	"users.password":       "Passwort",
	"users.minPassword":    "Mindestens 8 Zeichen",
	"users.role":           "Rolle",
	"users.delete":         "L\u00f6schen",
	"users.confirmDelete":  "Sind Sie sicher, dass Sie diesen Benutzer l\u00f6schen m\u00f6chten?",
	"users.setNewPassword": "Neues Passwort setzen",
	"users.setNewEmail":    "Neue E-Mail setzen",
	"users.viewProfile":    "Profil",
	"users.set":            "Setzen",
	"users.participant":    "Teilnehmer",
	"users.organizer":      "Organisator",
	"users.admin":          "Admin",

	// Settings
	"settings.emailConfig":          "E-Mail-Konfiguration",
	"settings.enableNotifications":  "E-Mail-Benachrichtigungen aktivieren",
	"settings.smtpServer":           "SMTP-Server",
	"settings.smtpHost":             "SMTP-Host",
	"settings.port":                 "Port",
	"settings.smtpUsername":         "SMTP-Benutzername",
	"settings.smtpPassword":         "SMTP-Passwort",
	"settings.sender":               "Absender",
	"settings.senderName":           "Absendername",
	"settings.senderEmail":          "Absender-E-Mail",
	"settings.notificationLimits":   "Benachrichtigungslimits",
	"settings.maxEmails":            "Max. E-Mails pro Tag (pro Benutzer)",
	"settings.reminderDays":         "Erinnerungstage f\u00fcr anstehende Eins\u00e4tze",
	"settings.saveSettings":         "Einstellungen speichern",
	"settings.testEmail":            "Test-E-Mail",
	"settings.testEmailDesc":        "Senden Sie eine Test-E-Mail an Ihre eigene Adresse, um die SMTP-Konfiguration zu \u00fcberpr\u00fcfen.",
	"settings.sendTestEmail":        "Test-E-Mail senden",
	"settings.howItWorks":           "So funktioniert es",
	"settings.howItWorksDesc":       "DutyRound sendet E-Mail-Benachrichtigungen automatisch im Hintergrund.",
	"settings.newOccurrence":        "Neuer Einsatz",
	"settings.newOccurrenceDesc":    "Teilnehmer erhalten eine \u00dcbersicht verf\u00fcgbarer Eins\u00e4tze mit offenen Pl\u00e4tzen.",
	"settings.unfilledParticipants": "Offene Pl\u00e4tze (Teilnehmer)",
	"settings.unfilledPartDesc":     "Teilnehmer werden \u00fcber anstehende Eins\u00e4tze benachrichtigt, die noch Leute brauchen.",
	"settings.unfilledOrganizers":   "Offene Pl\u00e4tze (Organisatoren)",
	"settings.unfilledOrgDesc":      "Organisatoren erhalten einen Hinweis \u00fcber anstehende Eins\u00e4tze mit freien Pl\u00e4tzen.",
	"settings.note":                 "Hinweis:",
	"settings.noteDesc":             "Admins erhalten keine Benachrichtigungs-E-Mails. Die Einstellung \u201eMax. E-Mails pro Tag\u201c verhindert Spam.",

	// Login
	"login.email":    "E-Mail",
	"login.password": "Passwort",
	"login.signIn":   "Anmelden",

	// Error page
	"error.notFound":       "Nicht gefunden",
	"error.pageNotFound":   "Seite nicht gefunden",
	"error.goHome":         "Zur Startseite",
	"error.somethingWrong": "Etwas ist schiefgelaufen",

	// Flash messages (handlers)
	"flash.invalidEmailOrPassword":  "Ung\u00fcltige E-Mail oder Passwort.",
	"flash.invalidFormData":         "Ung\u00fcltige Formulardaten.",
	"flash.failedCreateOccurrence":  "Einsatz konnte nicht erstellt werden.",
	"flash.occurrenceCreated":       "Einsatz erstellt.",
	"flash.failedUpdateOccurrence":  "Einsatz konnte nicht aktualisiert werden.",
	"flash.occurrenceUpdated":       "Einsatz aktualisiert.",
	"flash.failedDeleteOccurrence":  "Einsatz konnte nicht gel\u00f6scht werden.",
	"flash.occurrenceDeleted":       "Einsatz gel\u00f6scht.",
	"flash.passwordTooShort":        "Passwort muss mindestens 8 Zeichen lang sein.",
	"flash.failedUpdatePassword":    "Passwort konnte nicht aktualisiert werden.",
	"flash.passwordUpdated":         "Passwort aktualisiert.",
	"flash.invalidDates":            "Ung\u00fcltige Daten.",
	"flash.endDateAfterStart":       "Enddatum muss nach dem Startdatum liegen.",
	"flash.oooConflict":             "Sie haben Teilnahmen in diesem Zeitraum zugewiesen.",
	"flash.nameRequired":            "Name ist erforderlich.",
	"flash.failedCreateGroup":       "Gruppe konnte nicht erstellt werden.",
	"flash.groupCreated":            "Gruppe erstellt.",
	"flash.failedDeleteGroup":       "Gruppe konnte nicht gel\u00f6scht werden.",
	"flash.groupDeleted":            "Gruppe gel\u00f6scht.",
	"flash.failedCreateUser":        "Benutzer konnte nicht erstellt werden.",
	"flash.userCreated":             "Benutzer erstellt.",
	"flash.invalidRole":             "Ung\u00fcltige Rolle ausgew\u00e4hlt.",
	"flash.failedSetPassword":       "Passwort konnte nicht gesetzt werden.",
	"flash.failedDeleteUser":        "Benutzer konnte nicht gel\u00f6scht werden.",
	"flash.userDeleted":             "Benutzer gel\u00f6scht.",
	"flash.cannotDeleteSelf":        "Sie k\u00f6nnen Ihr eigenes Konto nicht l\u00f6schen.",
	"flash.failedSaveSettings":      "Einstellungen konnten nicht gespeichert werden.",
	"flash.settingsSaved":           "E-Mail-Einstellungen gespeichert.",
	"flash.failedSendTestEmail":     "Test-E-Mail konnte nicht gesendet werden: ",
	"flash.testEmailSent":           "Test-E-Mail gesendet an ",
	"flash.userOOOSignup":           "Sie sind an diesem Datum abwesend.",
	"flash.userOOOAssign":           "Benutzer ist an diesem Datum abwesend.",
	"flash.dateCantBePast":          "Das Datum darf nicht in der Vergangenheit liegen.",
	"flash.titleRequired":           "Titel ist erforderlich.",
	"flash.participantsMin":         "Teilnehmer m\u00fcssen mindestens 1 sein.",
	"flash.minExceedsMax":           "Mindestteilnehmer darf Maximum nicht \u00fcberschreiten.",
	"flash.occurrenceFull":          "Dieser Einsatz ist voll.",
	"flash.emailAlreadyInUse":       "Diese E-Mail-Adresse wird bereits verwendet.",
	"flash.failedUpdateEmail":       "E-Mail-Adresse konnte nicht aktualisiert werden.",
	"flash.emailUpdated":            "E-Mail-Adresse aktualisiert.",
	"flash.commentTooLong":          "Kommentar darf maximal 1000 Zeichen lang sein.",
	"flash.oooConflictDetail":       "Sie sind f\u00fcr einen oder mehrere Eins\u00e4tze in diesem Zeitraum eingetragen. Bitte tragen Sie sich zuerst aus.",

	// Page titles
	"title.dashboard":      "Dashboard",
	"title.occurrences":    "Eins\u00e4tze",
	"title.newOccurrence":  "Neuer Einsatz",
	"title.editOccurrence": "Einsatz bearbeiten",
	"title.calendar":       "Kalender",
	"title.leaderboard":    "Rangliste",
	"title.groups":         "Gruppen",
	"title.users":          "Benutzer",
	"title.settings":       "E-Mail-Einstellungen",
	"title.profile":        "Profil",

	// Search results
	"search.noResults":   "Keine Ergebnisse f\u00fcr",
	"search.people":      "Personen",
	"search.occurrences": "Eins\u00e4tze",

	// Occurrence over-limit
	"occ.full":                "Voll",
	"occ.overLimitWarning":    "Dieser Einsatz ist voll. Eine Anmeldung w\u00fcrde das Limit \u00fcberschreiten.",
	"occ.signupOverLimit":     "Eintragen (\u00fcber Limit)",
	"form.allowOverLimit":     "\u00dcber-Limit-Anmeldungen erlauben",

	// Language
	"lang.switch": "EN",

	// Relative date
	"rel.today":     "Heute",
	"rel.yesterday": "Gestern",
	"rel.tomorrow":  "Morgen",
	"rel.inDays":    "in %d Tagen",
	"rel.daysAgo":   "vor %d Tagen",

	// Dashboard extra
	"dash.yourSchedule":  "Meine Eins\u00e4tze",
	"dash.noUpcoming":    "Keine bevorstehenden Eins\u00e4tze.",
	"dash.browseSpots":   "Offene Pl\u00e4tze ansehen",

	// Occurrences list extra
	"occ.upcoming":  "Bevorstehend",
	"occ.allGroups": "Alle Gruppen",
	"occ.listView":  "Liste",
	"occ.cardView":  "Karten",

	// Occurrence form extra
	"form.copyFromExisting":      "Als Vorlage verwenden",
	"form.copyFromExistingLabel": "Titel, Beschreibung, Uhrzeit &amp; Teilnehmerlimits aus einem anderen Einsatz \u00fcbernehmen",
	"form.copySearchPlaceholder": "Einsatz suchen\u2026",

	// Comments
	"comment.title":         "Kommentare",
	"comment.none":          "Noch keine Kommentare.",
	"comment.placeholder":   "Kommentar hinzuf\u00fcgen\u2026",
	"comment.post":          "Senden",
	"comment.delete":        "l\u00f6schen",
	"comment.confirmDelete": "Diesen Kommentar l\u00f6schen?",

	// Email content
	"email.hi":                     "Hallo",
	"email.newOccAvailable":        "Neue Eins\u00e4tze sind verf\u00fcgbar. Tragen Sie sich ein, wenn Sie Zeit haben!",
	"email.headerOccurrence":       "Einsatz",
	"email.headerDate":             "Datum",
	"email.headerSignedUp":         "Eingetragen",
	"email.headerSpots":            "Pl\u00e4tze",
	"email.headerPeopleNeeded":     "Personen ben\u00f6tigt",
	"email.headerDaysUntil":        "Tage bis",
	"email.headerStillNeeded":      "Noch ben\u00f6tigt",
	"email.headerCurrent":          "Aktuell",
	"email.left":                   "\u00fcbrig",
	"email.needed":                 "ben\u00f6tigt",
	"email.days":                   "Tage",
	"email.unfilledParticipantMsg": "Die folgenden anstehenden Eins\u00e4tze brauchen noch Leute. \u00dcbernehmen Sie einen, wenn Sie Zeit haben!",
	"email.unfilledOrganizerMsg":   "Die folgenden anstehenden Eins\u00e4tze haben noch freie Pl\u00e4tze, die besetzt werden m\u00fcssen.",
	"email.testSubject":            "DutyRound - Test-E-Mail",
	"email.testTitle":              "Test-E-Mail",
	"email.testBody":               "Wenn Sie dies lesen, funktioniert Ihre DutyRound E-Mail-Konfiguration korrekt.",
	"email.testVerified":           "SMTP-Konfiguration best\u00e4tigt",
	"email.footer":                 "Dies ist eine automatische Benachrichtigung von DutyRound. Sie k\u00f6nnen Ihre Benachrichtigungseinstellungen bei Ihrem Administrator verwalten.",
	"email.subjectNew":             "Neue Eins\u00e4tze verf\u00fcgbar - DutyRound",
	"email.subjectUnfilledPart":    "Anstehende Eins\u00e4tze brauchen Leute - DutyRound",
	"email.subjectUnfilledOrg":     "Anstehende Eins\u00e4tze haben noch freie Pl\u00e4tze - DutyRound",
}
