export default function SettingsPage() {
  return (
    <div>
      <h1>Settings</h1>
      <form>
        <div>
          <label htmlFor="theme">Theme</label>
          <select id="theme">
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </select>
        </div>
        <div>
          <label htmlFor="notifications">Email Notifications</label>
          <input type="checkbox" id="notifications" />
        </div>
        <button type="submit">Save</button>
      </form>
    </div>
  );
}
