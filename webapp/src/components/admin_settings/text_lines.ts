// ConfigMessage's text fields (Message, AttachmentMessage,
// ActionSuccessfulMessage) are stored as string[] (one Go slice element per
// line) and joined with "\n" at render time (server/welcomebot.go). A plain
// textarea is the natural editor for that, so these convert between the two
// shapes symmetrically.
export function linesToText(lines: string[]): string {
    return lines.join('\n');
}

export function textToLines(text: string): string[] {
    return text.split('\n');
}
